package client

import (
	"fmt"
	"github.com/jcmturner/gokrb5/types"
	"os"
	"strings"
	"time"
)

// Client ticket cache.
type Cache struct {
	Entries map[string]CacheEntry
}

// Ticket cache entry.
type CacheEntry struct {
	Ticket     types.Ticket
	AuthTime   time.Time
	EndTime    time.Time
	RenewTill  time.Time
	SessionKey types.EncryptionKey
}

// Create a new client ticket cache.
func NewCache() *Cache {
	return &Cache{
		Entries: map[string]CacheEntry{},
	}
}

// Get a cache entry that matches the SPN.
func (c *Cache) GetEntry(spn string) (CacheEntry, bool) {
	e, ok := (*c).Entries[spn]
	return e, ok
}

// Add a ticket to the cache.
func (c *Cache) AddEntry(tkt types.Ticket, authTime, endTime, renewTill time.Time, sessionKey types.EncryptionKey) CacheEntry {
	spn := strings.Join(tkt.SName.NameString, "/")
	(*c).Entries[spn] = CacheEntry{
		Ticket:     tkt,
		AuthTime:   authTime,
		EndTime:    endTime,
		RenewTill:  renewTill,
		SessionKey: sessionKey,
	}
	return c.Entries[spn]
}

// Remove the cache entry for the defined SPN.
func (c *Cache) RemoveEntry(spn string) {
	delete(c.Entries, spn)
}

// Get a ticket from the cache for the SPN.
// Only a ticket that is currently valid will be returned.
func (cl *Client) GetCachedTicket(spn string) (types.Ticket, bool) {
	if e, ok := cl.Cache.GetEntry(spn); ok {
		//If within time window of ticket return it
		if time.Now().After(e.AuthTime) && time.Now().Before(e.EndTime) {
			return e.Ticket, true
		} else if time.Now().Before(e.RenewTill) {
			e, err := cl.RenewTicket(e)
			if err != nil {
				return e.Ticket, false
			}
			return e.Ticket, true
		}
	}
	var tkt types.Ticket
	return tkt, false
}

// Renew a cache entry ticket
func (cl *Client) RenewTicket(e CacheEntry) (CacheEntry, error) {
	spn := e.Ticket.SName
	_, tgsRep, err := cl.TGSExchange(spn, e.Ticket, e.SessionKey, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Renew err: %+v\n", err)
		return e, err
	}
	e = cl.Cache.AddEntry(
		tgsRep.Ticket,
		tgsRep.DecryptedEncPart.AuthTime,
		tgsRep.DecryptedEncPart.EndTime,
		tgsRep.DecryptedEncPart.RenewTill,
		tgsRep.DecryptedEncPart.Key,
	)
	return e, nil
}