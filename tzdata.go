package moexoptcalc

// Embed the timezone database so DateTime's LoadLocation("Europe/Moscow")
// resolves on any host, including minimal images (scratch, distroless) with no
// system tzdata. MOEX sends Moscow-local timestamps with no offset, and a
// correct instant needs the real zone (see DateTime in params.go).
//
// Trade-off: this import adds ~450 KB to every binary linking this package.
// Isolated here so the cost and reason are obvious in one place.
import _ "time/tzdata"
