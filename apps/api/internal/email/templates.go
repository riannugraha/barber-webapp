package email

import (
	"fmt"
	"html"
	"strings"
	"time"

	"flowbook/api/internal/db"
)

// format helpers
func formatPrice(cents int32) string {
	if cents == 0 {
		return "Gratis"
	}
	// Rp 85.000 style with dot separator
	s := fmt.Sprintf("%d", cents)
	// Insert dots every 3 from right
	var out strings.Builder
	for i, c := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			out.WriteString(".")
		}
		out.WriteRune(c)
	}
	return "Rp " + out.String()
}

func formatTimeLocal(t time.Time, tz string) string {
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Jakarta")
		tz = "Asia/Jakarta"
	}
	local := t.In(loc)
	return local.Format("Monday, 02 Jan 2006 15:04") + " " + tz
}

func formatTimeUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// BookingConfirmedData holds rendering data
func renderBookingConfirmedHTML(booking db.Booking, service db.Service, staff db.Staff, org db.Organization) (subject, htmlBody string) {
	tz := org.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	when := formatTimeLocal(booking.StartAt, tz)
	price := formatPrice(service.PriceCents)
	trackURL := fmt.Sprintf("/track/%s", booking.ID.String())
	subject = fmt.Sprintf("Booking Confirmed — %s • %s", service.Name, when)
	// Keep HTML simple but branded violet 260
	htmlBody = fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;line-height:1.6;color:#0f172a;max-width:600px;margin:auto;padding:24px">
  <div style="border:1px solid #e2e8f0;border-radius:12px;overflow:hidden">
    <div style="background:oklch(0.62 0.19 260);color:white;padding:20px 24px">
      <h1 style="margin:0;font-size:20px">Booking Confirmed ✓</h1>
      <p style="margin:6px 0 0;opacity:0.9">%s — %s</p>
    </div>
    <div style="padding:24px">
      <p>Hi %s,</p>
      <p>Booking kamu sudah <strong>CONFIRMED</strong>. Detail:</p>
      <table style="width:100%%;border-collapse:collapse;margin:16px 0">
        <tr><td style="padding:8px 0;color:#64748b;width:120px">Layanan</td><td><strong>%s</strong> (%d min + %d min buffer) — %s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Staff</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Waktu</td><td><strong>%s</strong><br/><span style="color:#64748b;font-size:12px">UTC %s • %s</span></td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Lokasi</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Booking ID</td><td style="font-family:monospace;font-size:12px">%s</td></tr>
      </table>
      <p style="margin:16px 0"><a href="%s" style="background:oklch(0.62 0.19 260);color:white;padding:10px 18px;border-radius:8px;text-decoration:none;display:inline-block">Lihat Booking →</a></p>
      <p style="color:#64748b;font-size:13px">File <code>booking.ics</code> terlampir — klik untuk tambah ke Google Calendar / Apple Calendar.</p>
      <hr style="border:none;border-top:1px solid #e2e8f0;margin:24px 0"/>
      <p style="color:#64748b;font-size:12px">Butuh reschedule? Buka halaman track atau balas email ini.<br/>%s</p>
    </div>
  </div>
</body>
</html>`,
		html.EscapeString(service.Name), html.EscapeString(when),
		html.EscapeString(booking.CustomerName),
		html.EscapeString(service.Name), service.DurationMinutes, service.BufferMinutes, html.EscapeString(price),
		html.EscapeString(staff.Name),
		html.EscapeString(when), html.EscapeString(formatTimeUTC(booking.StartAt)), html.EscapeString(tz),
		html.EscapeString(org.Name),
		html.EscapeString(booking.ID.String()),
		html.EscapeString(trackURL),
		html.EscapeString(org.Name),
	)
	return subject, htmlBody
}

func renderCancelledHTML(booking db.Booking, service db.Service, staff db.Staff, org db.Organization) (subject, htmlBody string) {
	tz := org.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	when := formatTimeLocal(booking.StartAt, tz)
	subject = fmt.Sprintf("Booking Cancelled — %s • %s", service.Name, when)
	htmlBody = fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;line-height:1.6;color:#0f172a;max-width:600px;margin:auto;padding:24px">
  <div style="border:1px solid #e2e8f0;border-radius:12px;overflow:hidden">
    <div style="background:#ef4444;color:white;padding:20px 24px">
      <h1 style="margin:0;font-size:20px">Booking Cancelled</h1>
      <p style="margin:6px 0 0;opacity:0.9">%s — %s</p>
    </div>
    <div style="padding:24px">
      <p>Hi %s,</p>
      <p>Booking kamu telah <strong>dibatalkan</strong>:</p>
      <table style="width:100%%;border-collapse:collapse;margin:16px 0">
        <tr><td style="padding:8px 0;color:#64748b;width:120px">Layanan</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Staff</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Waktu</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Booking ID</td><td style="font-family:monospace;font-size:12px">%s</td></tr>
      </table>
      <p>Jika ini tidak sengaja, silakan booking ulang di <a href="/book">/book</a>.</p>
      <p style="color:#64748b;font-size:12px">Jika sudah bayar, refund akan diproses sesuai kebijakan.</p>
      <hr style="border:none;border-top:1px solid #e2e8f0;margin:24px 0"/>
      <p style="color:#64748b;font-size:12px">%s</p>
    </div>
  </div>
</body>
</html>`,
		html.EscapeString(service.Name), html.EscapeString(when),
		html.EscapeString(booking.CustomerName),
		html.EscapeString(service.Name),
		html.EscapeString(staff.Name),
		html.EscapeString(when),
		html.EscapeString(booking.ID.String()),
		html.EscapeString(org.Name),
	)
	return subject, htmlBody
}

func renderReminderHTML(booking db.Booking, service db.Service, staff db.Staff, org db.Organization) (subject, htmlBody string) {
	tz := org.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	when := formatTimeLocal(booking.StartAt, tz)
	subject = fmt.Sprintf("Reminder H-1 — %s besok %s", service.Name, when)
	htmlBody = fmt.Sprintf(`<!doctype html>
<html>
<body style="font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;line-height:1.6;color:#0f172a;max-width:600px;margin:auto;padding:24px">
  <div style="border:1px solid #e2e8f0;border-radius:12px;overflow:hidden">
    <div style="background:oklch(0.62 0.19 260);color:white;padding:20px 24px">
      <h1 style="margin:0;font-size:20px">Reminder — 1 Jam Lagi ⏰</h1>
      <p style="margin:6px 0 0;opacity:0.9">%s dengan %s</p>
    </div>
    <div style="padding:24px">
      <p>Hi %s,</p>
      <p>Booking kamu <strong>1 jam lagi</strong>:</p>
      <table style="width:100%%;border-collapse:collapse;margin:16px 0">
        <tr><td style="padding:8px 0;color:#64748b;width:120px">Layanan</td><td><strong>%s</strong> (%d min)</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Staff</td><td>%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Waktu</td><td><strong>%s</strong></td></tr>
        <tr><td style="padding:8px 0;color:#64748b">Lokasi</td><td>%s</td></tr>
      </table>
      <p>Mohon datang 10 menit lebih awal. Jika perlu reschedule, buka <a href="/track/%s">/track/%s</a>.</p>
      <p style="color:#64748b;font-size:12px">Sampai jumpa! — %s</p>
    </div>
  </div>
</body>
</html>`,
		html.EscapeString(service.Name), html.EscapeString(staff.Name),
		html.EscapeString(booking.CustomerName),
		html.EscapeString(service.Name), service.DurationMinutes,
		html.EscapeString(staff.Name),
		html.EscapeString(when),
		html.EscapeString(org.Name),
		html.EscapeString(booking.ID.String()), html.EscapeString(booking.ID.String()),
		html.EscapeString(org.Name),
	)
	return subject, htmlBody
}
