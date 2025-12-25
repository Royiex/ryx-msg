package backend

import (
	"context"
	"fmt"
	"io"

	"berty.tech/weshnet/v2"
	"berty.tech/weshnet/v2/pkg/protocoltypes"
	"github.com/mr-tron/base58"
	"google.golang.org/protobuf/proto"
)

// Client wraps the weshnet service client
type Client struct {
	ctx    context.Context
	cancel context.CancelFunc
	svc    weshnet.ServiceClient
}

// NewClient creates a new backend client with the given data directory
func NewClient(dataDir string) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	svc, err := weshnet.NewPersistentServiceClient(dataDir)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Client{
		ctx:    ctx,
		cancel: cancel,
		svc:    svc,
	}, nil
}

// Close shuts down the client
func (c *Client) Close() error {
	c.cancel()
	return c.svc.Close()
}

// GetID returns the encoded contact ID for this client
func (c *Client) GetID() (string, error) {
	binaryContact, err := c.svc.ShareContact(c.ctx, &protocoltypes.ShareContact_Request{})
	if err != nil {
		return "", err
	}
	return base58.Encode(binaryContact.EncodedContact), nil
}

// Send sends a message to the specified contact
func (c *Client) Send(encodedContact string, msg string) error {
	contactBinary, err := base58.Decode(encodedContact)
	if err != nil {
		return err
	}
	contact, err := c.svc.DecodeContact(c.ctx,
		&protocoltypes.DecodeContact_Request{
			EncodedContact: contactBinary,
		})
	if err != nil {
		return err
	}

	_, err = c.svc.ContactRequestSend(c.ctx,
		&protocoltypes.ContactRequestSend_Request{
			Contact: contact.Contact,
		})
	if err != nil {
		return err
	}

	groupInfo, err := c.svc.GroupInfo(c.ctx, &protocoltypes.GroupInfo_Request{
		ContactPk: contact.Contact.Pk,
	})
	if err != nil {
		return err
	}

	_, err = c.svc.ActivateGroup(c.ctx, &protocoltypes.ActivateGroup_Request{
		GroupPk: groupInfo.Group.PublicKey,
	})
	if err != nil {
		return err
	}

	if err := c.waitForGroupReady(groupInfo.Group.PublicKey); err != nil {
		return err
	}

	_, err = c.svc.AppMessageSend(c.ctx, &protocoltypes.AppMessageSend_Request{
		GroupPk: groupInfo.Group.PublicKey,
		Payload: []byte(msg),
	})
	return err
}


// Receive waits for and returns an incoming message
func (c *Client) Receive() (string, error) {
	request, err := c.receiveContactRequest()
	if err != nil {
		return "", err
	}
	if request == nil {
		return "", fmt.Errorf("did not receive contact request")
	}

	_, err = c.svc.ContactRequestAccept(c.ctx, &protocoltypes.ContactRequestAccept_Request{ContactPk: request.ContactPk})
	if err != nil {
		return "", err
	}

	groupInfo, err := c.svc.GroupInfo(c.ctx, &protocoltypes.GroupInfo_Request{ContactPk: request.ContactPk})
	if err != nil {
		return "", err
	}

	_, err = c.svc.ActivateGroup(c.ctx, &protocoltypes.ActivateGroup_Request{GroupPk: groupInfo.Group.PublicKey})
	if err != nil {
		return "", err
	}

	if err := c.waitForGroupReady(groupInfo.Group.PublicKey); err != nil {
		return "", err
	}

	message, err := c.receiveMessage(groupInfo)
	if err != nil {
		return "", err
	}

	if message == nil {
		return "", fmt.Errorf("end of stream without receiving message")
	}

	return string(message.Message), nil
}

func (c *Client) receiveContactRequest() (*protocoltypes.AccountContactRequestIncomingReceived, error) {
	config, err := c.svc.ServiceGetConfiguration(c.ctx, &protocoltypes.ServiceGetConfiguration_Request{})
	if err != nil {
		return nil, err
	}

	subCtx, subCancel := context.WithCancel(c.ctx)
	defer subCancel()
	subMetadata, err := c.svc.GroupMetadataList(subCtx, &protocoltypes.GroupMetadataList_Request{
		GroupPk: config.AccountGroupPk,
	})
	if err != nil {
		return nil, err
	}

	for {
		metadata, err := subMetadata.Recv()
		if err == io.EOF || subMetadata.Context().Err() != nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		if metadata == nil || metadata.Metadata.EventType !=
			protocoltypes.EventType_EventTypeAccountContactRequestIncomingReceived {
			continue
		}

		request := &protocoltypes.AccountContactRequestIncomingReceived{}
		if err = proto.Unmarshal(metadata.Event, request); err != nil {
			return nil, err
		}

		return request, nil
	}
}

func (c *Client) receiveMessage(groupInfo *protocoltypes.GroupInfo_Reply) (*protocoltypes.GroupMessageEvent, error) {
	subCtx, subCancel := context.WithCancel(c.ctx)
	defer subCancel()
	subMessages, err := c.svc.GroupMessageList(subCtx, &protocoltypes.GroupMessageList_Request{
		GroupPk: groupInfo.Group.PublicKey,
	})
	if err != nil {
		return nil, err
	}

	for {
		message, err := subMessages.Recv()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return message, nil
	}
}

func (c *Client) waitForGroupReady(groupPk []byte) error {
	sub, err := c.svc.GroupMetadataList(c.ctx, &protocoltypes.GroupMetadataList_Request{
		GroupPk: groupPk,
	})
	if err != nil {
		return err
	}

	for {
		ev, err := sub.Recv()
		if err != nil {
			return err
		}
		if ev == nil || ev.Metadata == nil {
			continue
		}

		switch ev.Metadata.EventType {
		case protocoltypes.EventType_EventTypeGroupMemberDeviceAdded:
			return nil
		}
	}
}
