package email

import (
	"fmt"
	"strings"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
)

// BuildICS generates RFC5545 iCalendar content for a booking.
// Times are stored UTC in DB and rendered as UTC Zulu per spec (DTSTART:UTC).
// ORGANIZER from resendFrom, ATTENDEE is customer.
func BuildICS(booking db.Booking, service db.Service, staff db.Staff, org db.Organization) string {
	// UID must be globally unique per booking
	uid := fmt.Sprintf("%s@flowbook.example.com", booking.ID.String())
	// DTSTAMP is now UTC
	now := time.Now().UTC().Format("20060102T150405Z")
	dtStart := booking.StartAt.UTC().Format("20060102T150405Z")
	dtEnd := booking.EndAt.UTC().Format("20060102T150405Z")
	// Escape text per RFC5545
	escape := func(s string) string {
		r := strings.ReplaceAll(s, "\\", "\\\\")
		r = strings.ReplaceAll(r, ";", "\\;")
		r = strings.ReplaceAll(r, ",", "\\,")
		r = strings.ReplaceAll(r, "\n", "\\n")
		return r
	}
	summary := escape(fmt.Sprintf("%s with %s", service.Name, staff.Name))
	location := escape(org.Name)
	if org.Name == "" {
		location = escape("FlowBarber Studio")
	}
	descriptionParts := []string{
		fmt.Sprintf("Booking %s", booking.ID.String()),
		fmt.Sprintf("Service: %s (%d min)", service.Name, service.DurationMinutes),
		fmt.Sprintf("Staff: %s", staff.Name),
		fmt.Sprintf("Customer: %s <%s>", booking.CustomerName, booking.CustomerEmail),
	}
	if booking.Notes.Valid && strings.TrimSpace(booking.Notes.String) != "" {
		descriptionParts = append(descriptionParts, fmt.Sprintf("Notes: %s", booking.Notes.String))
	}
	descriptionParts = append(descriptionParts, fmt.Sprintf("View: /track/%s", booking.ID.String()))
	description := escape(strings.Join(descriptionParts, "\\n"))

	// Status mapping: CONFIRMED vs others
	status := "CONFIRMED"
	if booking.Status == "CANCELLED" {
		status = "CANCELLED"
	}

	// ORGANIZER from org or default
	organizer := "noreply@flowbook.example.com"
	if strings.Contains(org.Name, "@") {
		organizer = org.Name
	}

	return strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//FlowBook//Booking//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:REQUEST",
		"BEGIN:VEVENT",
		fmt.Sprintf("UID:%s", uid),
		fmt.Sprintf("DTSTAMP:%s", now),
		fmt.Sprintf("DTSTART:%s", dtStart),
		fmt.Sprintf("DTEND:%s", dtEnd),
		fmt.Sprintf("SUMMARY:%s", summary),
		fmt.Sprintf("DESCRIPTION:%s", description),
		fmt.Sprintf("LOCATION:%s", location),
		fmt.Sprintf("STATUS:%s", status),
		fmt.Sprintf("ORGANIZER;CN=%s:mailto:%s", escape(org.Name), organizer),
		fmt.Sprintf("ATTENDEE;CN=%s;RSVP=TRUE:mailto:%s", escape(booking.CustomerName), booking.CustomerEmail),
		fmt.Sprintf("SEQUENCE:%d", 0),
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n")
}

// BuildICSUID returns deterministic UID for a booking (for testing).
func BuildICSUID(bookingID uuid.UUID) string {
	return fmt.Sprintf("%s@flowbook.example.com", bookingID.String())
}
