package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

// Resolves a counterparty from the routing information and validates in the incoming
// counterparty. If there is an error, this function will return the appropriate error
// response and status code to the client so the caller simply has to terminate handling.
func (s *Server) ResolveCounterparty(c *gin.Context, in *api.Routing) (vasp *models.Counterparty, err error) {
	if err = in.Validate(); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return nil, err
	}

	var protocol enum.Protocol
	if protocol, err = enum.ParseProtocol(in.Protocol); err != nil {
		// NOTE: if this error occurs, it means that the Validate code above has a bug in it.
		c.Error(fmt.Errorf("could not parse protocol from valid routing API request: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not identify counterparty from routing information"))
		return nil, err
	}

	ctx := c.Request.Context()

	// Ideally we look up the counterparty by ID for any protocol
	if !in.CounterpartyID.IsZero() {
		if vasp, err = s.store.RetrieveCounterparty(ctx, in.CounterpartyID); err != nil {
			if errors.Is(err, dberr.ErrNotFound) {
				c.JSON(http.StatusNotFound, api.Error("could not identify counterparty from routing information"))
				return nil, err
			}

			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not identify counterparty from routing information"))
			return nil, err
		}
	} else {
		// If there was no counterparty ID given, then try the backup methods for lookup on a per-protocol basis
		switch protocol {
		case enum.ProtocolTRISA:
			// Lookup the counterparty by travel address
			if vasp, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
				// NOTE: CounterpartyFromTravelAddress handles API response back to user.
				return nil, err
			}

		case enum.ProtocolTRP:
			// Lookup the counterparty by travel address
			if vasp, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
				// NOTE: CounterpartyFromTravelAddress handles API response back to user.
				return nil, err
			}

		case enum.ProtocolSunrise:
			// Get or create the counterparty for the associated email address and/or name
			if vasp, err = s.store.GetOrCreateSunriseCounterparty(ctx, in.EmailAddress, in.Counterparty); err != nil {
				c.Error(err)
				c.JSON(http.StatusConflict, api.Error("could not find or create a counterparty with the specified name and/or email address"))
				return nil, err
			}

		default:
			// If we get here the protocol is valid but not handled, so this is a developer
			// bug that we need to fix ASAP, hence the panic.
			panic(fmt.Errorf("unhandled protocol in resolve counterparty: %q", protocol.String()))
		}
	}

	switch {
		// If vasp is nil we probably shouldn't have made it this far in the code, but protecting ourselves anyway.
		case vasp == nil:
			c.JSON(http.StatusNotFound, api.Error("could not identify counterparty from routing information"))
			return nil, errors.New("unhandled nil vasp at end of resolve counterparty")
		case vasp.Protocol != protocol:
			err := errors.New("could not find counterparty that supports requested protocol")
			c.JSON(http.StatusNotFound, api.Error(err)
			return nil, err
		default:
		return vasp, nil
	}
}

func (s *Server) CounterpartyFromTravelAddress(c *gin.Context, address string) (cp *models.Counterparty, err error) {
	var (
		dst    string
		dstURI *traddr.URL
	)

	if dst, err = traddr.Decode(address); err != nil {
		c.Error(fmt.Errorf("could not decode travel address %q: %w", address, err))
		c.JSON(http.StatusBadRequest, api.Error("could not parse the travel address"))
		return nil, err
	}

	if dstURI, err = traddr.Parse(dst); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address url"))
		return nil, err
	}

	if cp, err = s.findCounterparty(c.Request.Context(), dstURI); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.Error(fmt.Errorf("could not identify counterparty for %s or %s", dstURI.Hostname(), dstURI.Host))
			c.JSON(http.StatusNotFound, api.Error("could not identify counterparty from travel address"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return nil, err
	}

	return cp, nil
}

func (s *Server) findCounterparty(ctx context.Context, uri *traddr.URL) (cp *models.Counterparty, err error) {
	// Lookup counterparty by hostname first (e.g. the common name).
	if cp, err = s.store.LookupCounterparty(ctx, models.FieldCommonName, uri.Hostname()); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			// If we couldn't find it, try again by endpoint
			// NOTE: this is primarily to assist with lookups for localhost where the
			// port number is the only differentiating aspect of the node.
			if cp, err = s.store.LookupCounterparty(ctx, models.FieldCommonName, uri.Host); err != nil {
				return nil, dberr.ErrNotFound
			}

			// Found! Short-circuit the error handling by returning early!
			return cp, err
		}

		// Return the internal error
		return nil, err
	}

	// Found on first try!
	return cp, nil
}
