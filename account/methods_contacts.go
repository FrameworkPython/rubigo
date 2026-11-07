package account

import (
	"context"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Contacts methods ================================================

// GetContacts returns the user's address book (paged).
func (a *Account) GetContacts(ctx context.Context, startID string) (*events.Event, error) {
	in := map[string]any{}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getContacts", Input: in, Encrypt: true})
}

// GetContactsUpdates returns the latest contacts updates.
func (a *Account) GetContactsUpdates(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getContactsUpdates", Input: map[string]any{}, Encrypt: true})
}

// AddAddressBook adds a single phone number to the address book.
func (a *Account) AddAddressBook(ctx context.Context, phone, firstName, lastName string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "addAddressBook",
		Input: map[string]any{
			"phone":      phone,
			"first_name": firstName,
			"last_name":  lastName,
		},
		Encrypt: true,
	})
}

// DeleteContact deletes a contact from the address book.
func (a *Account) DeleteContact(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "deleteContact", Input: map[string]any{"user_guid": userGUID}, Encrypt: true})
}

// ResetContacts resets the address book.
func (a *Account) ResetContacts(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "resetContacts", Input: map[string]any{}, Encrypt: true})
}
