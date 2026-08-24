import type { Booking, Service } from "./api";

function toICSDate(iso: string): string {
  const d = new Date(iso);
  // UTC format YYYYMMDDTHHMMSSZ
  const pad = (n: number) => String(n).padStart(2, "0");
  return (
    d.getUTCFullYear().toString() +
    pad(d.getUTCMonth() + 1) +
    pad(d.getUTCDate()) +
    "T" +
    pad(d.getUTCHours()) +
    pad(d.getUTCMinutes()) +
    pad(d.getUTCSeconds()) +
    "Z"
  );
}

export function buildICS(booking: Booking, service?: Service): string {
  const now = toICSDate(new Date().toISOString());
  const dtStart = toICSDate(booking.startAt);
  const dtEnd = toICSDate(booking.endAt);
  const summary = service ? `${service.name} — FlowBarber Studio` : "Booking FlowBarber Studio";
  const description = `Booking ${booking.id}\\nStaff: ${booking.staffId}\\nCustomer: ${booking.customerName} (${booking.customerEmail})\\n${booking.notes ?? ""}`.replace(/\n/g, "\\n");
  const location = "FlowBarber Studio, Jakarta";
  const uid = `${booking.id}@flowbook.example.com`;

  return [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//FlowBook//Booking//ID",
    "CALSCALE:GREGORIAN",
    "METHOD:PUBLISH",
    "BEGIN:VEVENT",
    `UID:${uid}`,
    `DTSTAMP:${now}`,
    `DTSTART:${dtStart}`,
    `DTEND:${dtEnd}`,
    `SUMMARY:${summary}`,
    `DESCRIPTION:${description}`,
    `LOCATION:${location}`,
    "STATUS:CONFIRMED",
    "END:VEVENT",
    "END:VCALENDAR",
  ].join("\r\n");
}

export function downloadICS(booking: Booking, service?: Service) {
  const ics = buildICS(booking, service);
  const blob = new Blob([ics], { type: "text/calendar;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `flowbook-${booking.id}.ics`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
