package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ParticipantType represents the type of participant (Member or EventHost)
type ParticipantType string

const (
	Member    ParticipantType = "Member"
	EventHost ParticipantType = "EventHost"
)

// TicketStatus represents the status of a ticket (Available, Sold, Resold, Used)
type TicketStatus string

const (
	Available TicketStatus = "Available"
	Sold      TicketStatus = "Sold"
	Resold    TicketStatus = "Resold"
	Used      TicketStatus = "Used"
)

// Participant defines the structure of a participant
type Participant struct {
	ID   string          `json:"ID"`
	Name string          `json:"name"`
	Type ParticipantType `json:"type"`
}

// Event defines the structure of an event
type Event struct {
	ID       string   `json:"ID"`
	Name     string   `json:"name"`
	HostID   string   `json:"hostID"`
	Date     string   `json:"date"`
	Location string   `json:"location"`
	Tickets  []string `json:"tickets"`
}

// Ticket defines the structure of a ticket
type Ticket struct {
	ID      string       `json:"ID"`
	EventID string       `json:"eventID"`
	Status  TicketStatus `json:"status"`
	Owner   string       `json:"owner"`
}

// ConcertTicketBookingChaincode is the chaincode implementation
type ConcertTicketBookingChaincode struct {
	contractapi.Contract
}

// Init initializes the chaincode
func (ctbc *ConcertTicketBookingChaincode) Init(ctx contractapi.TransactionContextInterface) error {
	// No initialization needed for this example
	fmt.Println("Concert Ticket Booking Chaincode initialized")
	return nil
}

// RegisterParticipant registers a participant as either a member or an event host
func (ctbc *ConcertTicketBookingChaincode) RegisterParticipant(ctx contractapi.TransactionContextInterface, participantID, name string, participantType ParticipantType) error {
	participant := &Participant{
		ID:   participantID,
		Name: name,
		Type: participantType,
	}

	// Save the participant to the world state
	err := ctx.GetStub().PutState(participantID, participantToBytes(participant))
	if err != nil {
		return fmt.Errorf("failed to put participant: %v", err)
	}

	fmt.Printf("Participant with ID '%s' registered as a '%s'\n", participantID, participantType)
	return nil
}

// CreateEvent creates a new event with the provided details
func (ctbc *ConcertTicketBookingChaincode) CreateEvent(ctx contractapi.TransactionContextInterface, eventID, eventName, hostID, eventDate, location string) error {
	event := &Event{
		ID:       eventID,
		Name:     eventName,
		HostID:   hostID,
		Date:     eventDate,
		Location: location,
		Tickets:  []string{},
	}

	// Save the event to the world state
	err := ctx.GetStub().PutState(eventID, eventToBytes(event))
	if err != nil {
		return fmt.Errorf("failed to put event: %v", err)
	}

	fmt.Printf("Event with ID '%s' created\n", eventID)
	return nil
}

// ListAvailableTickets lists all available tickets for the given event
func (ctbc *ConcertTicketBookingChaincode) ListAvailableTickets(ctx contractapi.TransactionContextInterface, eventID string) ([]string, error) {
	eventBytes, err := ctx.GetStub().GetState(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to read event %s: %v", eventID, err)
	}
	if eventBytes == nil {
		return nil, fmt.Errorf("event %s does not exist", eventID)
	}

	event := &Event{}
	err = bytesToEvent(eventBytes, event)
	if err != nil {
		return nil, err
	}

	availableTickets := []string{}
	for _, ticketID := range event.Tickets {
		ticketBytes, err := ctx.GetStub().GetState(ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to read ticket %s: %v", ticketID, err)
		}

		ticket := &Ticket{}
		err = bytesToTicket(ticketBytes, ticket)
		if err != nil {
			return nil, err
		}

		if ticket.Status == Available {
			availableTickets = append(availableTickets, ticket.ID)
		}
	}

	return availableTickets, nil
}

// SellTicket sells a ticket to the given participant for the specified event
func (ctbc *ConcertTicketBookingChaincode) SellTicket(ctx contractapi.TransactionContextInterface, ticketID, participantID string) error {
	ticketBytes, err := ctx.GetStub().GetState(ticketID)
	if err != nil {
		return fmt.Errorf("failed to read ticket %s: %v", ticketID, err)
	}
	if ticketBytes == nil {
		return fmt.Errorf("ticket %s does not exist", ticketID)
	}

	ticket := &Ticket{}
	err = bytesToTicket(ticketBytes, ticket)
	if err != nil {
		return err
	}

	if ticket.Status != Available {
		return fmt.Errorf("ticket %s is not available for sale", ticketID)
	}

	ticket.Status = Sold
	ticket.Owner = participantID

	// Update the ticket in the world state
	err = ctx.GetStub().PutState(ticketID, ticketToBytes(ticket))
	if err != nil {
		return fmt.Errorf("failed to update ticket: %v", err)
	}

	fmt.Printf("Ticket with ID '%s' sold to participant with ID '%s'\n", ticketID, participantID)
	return nil
}

// ResellTicket resells a ticket from the current owner to the specified participant
func (ctbc *ConcertTicketBookingChaincode) ResellTicket(ctx contractapi.TransactionContextInterface, ticketID, newOwnerID string) error {
	ticketBytes, err := ctx.GetStub().GetState(ticketID)
	if err != nil {
		return fmt.Errorf("failed to read ticket %s: %v", ticketID, err)
	}
	if ticketBytes == nil {
		return fmt.Errorf("ticket %s does not exist", ticketID)
	}

	ticket := &Ticket{}
	err = bytesToTicket(ticketBytes, ticket)
	if err != nil {
		return err
	}

	if ticket.Status != Sold {
		return fmt.Errorf("ticket %s is not sold and cannot be resold", ticketID)
	}

	ticket.Status = Resold
	ticket.Owner = newOwnerID

	// Update the ticket in the world state
	err = ctx.GetStub().PutState(ticketID, ticketToBytes(ticket))
	if err != nil {
		return fmt.Errorf("failed to update ticket: %v", err)
	}

	fmt.Printf("Ticket with ID '%s' resold to participant with ID '%s'\n", ticketID, newOwnerID)
	return nil
}

// UseTicket marks a ticket as used for the specified event
func (ctbc *ConcertTicketBookingChaincode) UseTicket(ctx contractapi.TransactionContextInterface, ticketID, eventID string) error {
	ticketBytes, err := ctx.GetStub().GetState(ticketID)
	if err != nil {
		return fmt.Errorf("failed to read ticket %s: %v", ticketID, err)
	}
	if ticketBytes == nil {
		return fmt.Errorf("ticket %s does not exist", ticketID)
	}

	ticket := &Ticket{}
	err = bytesToTicket(ticketBytes, ticket)
	if err != nil {
		return err
	}

	if ticket.Status != Sold && ticket.Status != Resold {
		return fmt.Errorf("ticket %s is not valid for entry", ticketID)
	}

	eventBytes, err := ctx.GetStub().GetState(eventID)
	if err != nil {
		return fmt.Errorf("failed to read event %s: %v", eventID, err)
	}
	if eventBytes == nil {
		return fmt.Errorf("event %s does not exist", eventID)
	}

	event := &Event{}
	err = bytesToEvent(eventBytes, event)
	if err != nil {
		return err
	}

	for _, ticketID := range event.Tickets {
		if ticketID == ticket.ID {
			ticket.Status = Used
			// Update the ticket in the world state
			err = ctx.GetStub().PutState(ticketID, ticketToBytes(ticket))
			if err != nil {
				return fmt.Errorf("failed to update ticket: %v", err)
			}

			fmt.Printf("Ticket with ID '%s' used for event with ID '%s'\n", ticketID, eventID)
			return nil
		}
	}

	return fmt.Errorf("ticket %s does not belong to event %s", ticketID, eventID)
}

// participantToBytes converts a participant to a byte array
func participantToBytes(participant *Participant) []byte {
	participantBytes, _ := json.Marshal(participant)
	return participantBytes
}

// bytesToParticipant converts a byte array to a participant
func bytesToParticipant(participantBytes []byte, participant *Participant) error {
	err := json.Unmarshal(participantBytes, participant)
	if err != nil {
		return fmt.Errorf("failed to unmarshal participant: %v", err)
	}
	return nil
}

// eventToBytes converts an event to a byte array
func eventToBytes(event *Event) []byte {
	eventBytes, _ := json.Marshal(event)
	return eventBytes
}

// bytesToEvent converts a byte array to an event
func bytesToEvent(eventBytes []byte, event *Event) error {
	err := json.Unmarshal(eventBytes, event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %v", err)
	}
	return nil
}

// ticketToBytes converts a ticket to a byte array
func ticketToBytes(ticket *Ticket) []byte {
	ticketBytes, _ := json.Marshal(ticket)
	return ticketBytes
}

// bytesToTicket converts a byte array to a ticket
func bytesToTicket(ticketBytes []byte, ticket *Ticket) error {
	err := json.Unmarshal(ticketBytes, ticket)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ticket: %v", err)
	}
	return nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&ConcertTicketBookingChaincode{})
	if err != nil {
		fmt.Printf("Error creating concert ticket booking chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting concert ticket booking chaincode: %s", err.Error())
	}
}
