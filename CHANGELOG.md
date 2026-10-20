# Changelog

Die vollständige Versionshistorie von EchoLox, einschließlich Prereleases. Historische Release-Notes dokumentieren den jeweiligen damaligen Stand. Für ältere Tags ohne GitHub-Release sind die Commit-Beschreibungen aufgeführt.

## [0.4.0](https://github.com/BattloXX/EchoLox/releases/tag/v0.4.0) — 2026-10-20

Stabiler Release mit allen Änderungen seit **0.3.2**, einschließlich der Verbesserungen aus **0.3.3-rc1 bis rc4**.

### Virtuelle Eingänge und Loxone-Befehle

- Virtuelle Eingänge lassen sich unter Discovery direkt über ihren exakten Namen prüfen. Die Prüfung liest nur den Status und sendet keinen Schaltbefehl.
- Die Oberfläche erklärt, dass die automatische Discovery App-Bedienelemente aus `LoxAPP3.json` findet und virtuelle Eingänge ohne App-Eintrag fehlen können. Ein fehlender Discovery-Eintrag verhindert die manuelle Zuordnung nicht.
- Zielnamen sind im Geräteformular bearbeitbar und bleiben beim Bearbeiten und Umbenennen erhalten. Der Alexa-Name darf vom Loxone-Ziel abweichen, beispielsweise „Küchenlicht“ → `echolox_licht_kueche`.
- Nach einer erfolgreichen Namensprüfung kann ein Gerät mit vorausgefülltem Ziel angelegt werden.
- HTTP-Befehle werten den Loxone-Ergebniscode aus. Abgelehnte Befehle mit HTTP 200 werden als Fehler erkannt; XML- und JSON-Antworten sowie numerische und textuelle Ergebnis-Codes werden berücksichtigt.
- HTTP-Impulse werden mit dem Webservice-Befehl `pulse` gesendet.
- Direkte Statusprüfungen verwenden den lesenden `/state`-Endpunkt und unterscheiden fehlende Eingänge von verweigertem Zugriff.

### Einrichtung und Bedienung

- Neue integrierte Hilfe mit vollständigem Küchenlicht-Beispiel, Erklärungen zu Ein/Aus, Impuls, Dimmer, Farblicht und Szenen sowie Hilfe bei typischen Problemen.
- Die Geräteseite führt durch die erste Einrichtung; bei einer leeren Geräteliste wird die Anleitung automatisch geöffnet.
- Verständliche Feldnamen, deutsche Gerätetypen und Erläuterungen der Zielwerte direkt im Formular.
- Verbindung prüfen, Eingangsnamen prüfen und tatsächlich Testwerte senden sind klar unterschieden. Die Oberfläche erklärt, dass Schaltbefehle angeschlossene Geräte oder Szenen auslösen können.
- Testergebnisse erklären, ob der Miniserver die Werte angenommen hat und was anschließend in der angeschlossenen Logik zu prüfen ist.
- Deutsche Statusanzeigen mit nächsten Schritten, zusätzlicher Filter für verweigerten Zugriff und hilfreiche Meldungen bei leeren Listen oder erfolgloser Suche.
- Rückmeldungen nach dem Speichern sowie verständlichere Meldungen bei Lade-, Verbindungs- und Speicherfehlern.
- Die gemeinsame Übertragungsart wird im Geräteformular aus den Einstellungen angezeigt. Ports, Alexa-Identität und MQTT sind in den Einstellungen aufklappbar; Hinweise erklären das Übernehmen nach einem Neustart.
- Die Bridge kann für Alexa erneut im Netzwerk bekannt gemacht werden. Die Oberfläche erklärt die Wirkung, ohne pauschal zum Löschen aller Alexa-Geräte aufzufordern.
- Hilfe ist über die Desktop- und Mobilnavigation erreichbar. Die Desktopnavigation kann bei knapper Breite horizontal gescrollt werden.
- README korrigiert: Für HTTP-Webservice-Befehle ist ein normaler virtueller Eingang vorgesehen. Ein virtueller HTTP-Eingang dient dem Abfragen externer Webseiten. Beispiele unterscheiden Ein/Aus und Impuls.

### Miniserver-Anmeldung

- LoxBerry-Zugangsdaten werden korrekt übernommen: vorhandene Raw-Felder haben Vorrang; andernfalls werden URI-kodierte Werte genau einmal dekodiert. Plus- und Prozentzeichen bleiben korrekt erhalten.
- Ungültige Kodierungen werden ohne Ausgabe der Zugangsdaten gemeldet.
- Der Verbindungstest liefert Fehler als gültiges JSON. Eine abgewiesene Anmeldung verursacht dadurch keinen JSON-Parserfehler mehr in der Oberfläche.

### LoxBerry-Installation und Dienststart

- Der HTTP-Port wird vor der LoxBerry-Loginitialisierung gebunden. Die Initialisierung läuft asynchron mit einem Zeitlimit von fünf Sekunden und blockiert den Start nicht mehr.
- Alle drei Linux-Binaries sind statisch gelinkt, einschließlich amd64; der Release-Workflow prüft die statische Linkung.
- Architekturangaben entsprechen `uname`: `x86_64`, `aarch64`, `armv7l`. Die Liste bleibt für LoxBerrys Konfigurationsparser in Anführungszeichen.
- Fehler beim Anlegen und Kopieren der Plugin-Icons während der Installation sind behoben.
- Die Plugin-Startseite liest die Konfiguration aus `data/plugins` und wertet den Port im Server-Abschnitt aus.
- Bei nicht erreichbarem Dienst zeigt die Plugin-Startseite eine Diagnoseseite mit Neustartmöglichkeit.
- Nach der Installation wird geprüft, ob der Dienstport lauscht. Bei Problemen werden Dienststatus und Journalmeldungen im Installationslog ausgegeben.

### Release und Changelog

- Versionsstände in Plugin-Metadaten, Makefile und Linux-Binaries auf **0.4.0** vereinheitlicht.
- Vollständiges `CHANGELOG.md` mit allen bisherigen GitHub-Releases und älteren Git-Tags; eigene Release-Notes für 0.4.0.
- Der Release-Workflow verwendet die ausführlichen Release-Notes und den konkreten Build-Commit als Tag-Ziel; das Installations-ZIP erhält eine SHA-256-Prüfsumme.
- Prereleases aktualisieren nur den Prerelease-Kanal. Stabile Releases aktualisieren beide LoxBerry-Updatekanäle.

### Installation und Aktualisierung

`EchoLox-0.4.0.zip` im LoxBerry Plugin Manager installieren oder das angebotene Plugin-Update verwenden. Das Paket enthält Binaries für amd64/x86-64, arm64 und armv7 sowie die integrierte Hilfe und das Changelog.

Bestehende Geräte können weiterverwendet werden. Für ein vorhandenes Ziel wie `echolox_licht_kueche` ein Gerät vom Typ **Schalter** mit **Ein/Aus** verwenden und im Feld **Ein und Aus** den exakten Zielnamen eintragen. Das Loxone-Programm muss den Eingang enthalten und auf den Miniserver übertragen sein.

### Validierung

Go-Tests, `go vet`, JavaScript-Syntaxprüfung, Shell- und PHP-Syntaxprüfungen sowie interne HTML-Verlinkungen geprüft. Linux-Binaries für alle drei Architekturen gebaut und statische Linkung kontrolliert. ZIP-Inhalt und SHA-256-Prüfsumme werden beim Release geprüft. Ein Funktionstest am echten LoxBerry/Miniserver steht noch aus.

**Alle Änderungen seit 0.3.2:** https://github.com/BattloXX/EchoLox/compare/v0.3.2...v0.4.0


## [v0.3.3-rc4](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.3-rc4) — 2026-10-20 — Prerelease

Testversion 0.3.3-rc4

- Virtuelle Eingänge lassen sich in Discovery gezielt per Name prüfen, ohne einen Schaltbefehl zu senden. Die App-Discovery kann Eingänge ohne App-Eintrag übersehen.
- Zielnamen sind im Geräteformular bearbeitbar und bleiben beim Bearbeiten erhalten, z.B. `echolox_licht_kueche` für das Alexa-Gerät „Küchenlicht“.
- Miniserver-Fehler werden auch bei HTTP 200 erkannt. Der HTTP-Impulsbefehl verwendet jetzt `pulse`.
- Neue Einrichtungshilfe mit Küchenlicht-Beispiel, verständlichen Feldnamen, Erklärungen der Schaltmodi, deutschen Statusmeldungen und konkreten nächsten Schritten.
- Verbindungsprüfung und Schaltbefehl sind klar unterschieden. Erweiterte Einstellungen sind aufklappbar.
- Enthält die Zugangsdaten- und JSON-Fehlerkorrekturen aus rc3 sowie statisch gelinkte Linux-Binaries für amd64, arm64 und armv7.

Validierung: Go-Tests, go vet, JavaScript-Syntax und interne HTML-Verlinkungen geprüft; alle drei Architekturen gebaut. Ein Test am echten LoxBerry/Miniserver steht noch aus.

Installation: `EchoLox-0.3.3-rc4.zip` im LoxBerry Plugin Manager installieren. Der stabile Updatekanal bleibt unverändert.

## [v0.3.3-rc3](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.3-rc3) — 2026-10-19 — Prerelease

Testversion 0.3.3-rc3 auf Basis von rc2.

- Korrigiert die Übernahme von Miniserver-Zugangsdaten aus LoxBerry: Raw-Felder werden bevorzugt, URL-kodierte Werte andernfalls genau einmal dekodiert. Sonderzeichen sowie Plus- und Prozentzeichen bleiben korrekt erhalten.
- „Miniserver testen“ liefert Fehler als gültiges JSON, sodass eine abgewiesene Anmeldung keinen „Unexpected token“-Fehler mehr erzeugt.
- Regressionstests prüfen die Anmeldung für Discovery, Verbindungstest, VI-Prüfung und Gerätebefehle sowie die JSON-Fehlerantworten.
- Enthält statisch gelinkte Linux-Binaries für amd64 (x86-64), arm64 und armv7.

Validierung: `go build ./...`, `go vet ./...` und `go test ./...` erfolgreich; alle drei Architekturen gebaut und statische Linkung geprüft. Ein Test am betroffenen LoxBerry/Miniserver steht noch aus.

Installation: `EchoLox-0.3.3-rc3.zip` im LoxBerry Plugin Manager installieren.

## [v0.3.3-rc2](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.3-rc2) — 2026-10-17 — Prerelease

Testversion aus PR #2 (ersetzt rc1):
- Behebt "wrong architecture" bei der Installation auf neueren LoxBerry-Versionen (ARCHITECTURE jetzt x86_64,aarch64,armv7l)
- HTTP-Port wird vor der LoxBerry-Log-Initialisierung gebunden (initlog.php mit 5s-Timeout)
- Icon-Installationsfehler behoben
- Alle Binaries statisch gelinkt
- Diagnoseseite im Plugin, falls EchoLox nicht erreichbar ist (inkl. Neustart-Button)
- Port-Check nach der Installation im Installationslog

## [v0.3.3-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.3-rc1) — 2026-10-17 — Prerelease

Testversion aus PR #2:
- HTTP-Port wird vor der LoxBerry-Log-Initialisierung gebunden (initlog.php mit 5s-Timeout) – behebt hängenden Start / "Verbindung abgelehnt"
- Icon-Installationsfehler behoben
- Alle Binaries statisch gelinkt
- Diagnoseseite im Plugin, falls EchoLox nicht erreichbar ist (inkl. Neustart-Button)
- Port-Check nach der Installation im Installationslog

## [v0.3.2-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.2-rc1) — 2026-08-08 — Prerelease

Build aus lokal kompilierten Bins

**Commit:** feat: LoxBerry-4 Log-Level-Integration, Header-Logo, Dokumentations-Link

## [v0.3.2](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.2) — 2026-08-08

Änderungen seit dem letzten Stable-Release **v0.2.10**:

#### LoxBerry 4
- Kompatibilität mit dem LoxBerry-4-Autoupdate-Mechanismus verifiziert
- Log-Level-Dropdown im Plugin Manager aktiviert — EchoLox spiegelt Info/Debug-Meldungen gefiltert nach dem eingestellten Level zusätzlich in LoxBerrys zentralen Log-Browser (**System > Log Files**); die eigene, detaillierte `/logs`-Oberfläche bleibt davon unabhängig
- „Dokumentation"-Button im Plugin Manager verlinkt jetzt auf dieses GitHub-Repo

#### Port 80 & Netzwerk
- Port-80-Bindung gehärtet: verpflichtendes `tcp4`-Binding + optionales `tcp6` (best effort), umgeht den bekannten LoxBerry-Apache-tcp6-Hänger
- Optionaler Reverse-Proxy-Modus zum Teilen von Port 80 mit LoxBerrys eigenem Apache, inkl. In-App-Selbstreparatur über die Einstellungsseite (kein SSH nötig, falls ein LoxBerry-Core-Update die Apache-Konfiguration überschreibt)
- IPv6-Kompatibilität: korrekte `host:port`-Bildung für IPv6-Miniserver-Adressen, Dual-Stack-HTTP-Server

#### Stabilität
- Alexa-Discovery-Cache: TTL-basiertes Aufräumen statt unbegrenztem Wachstum
- SSDP-Antworten: Rate-Limiting + begrenzter Worker-Pool statt unbegrenzter Goroutines
- Kein stiller MQTT→HTTP-Fallback mehr bei nicht unterstütztem Transport

#### Web-Oberfläche
- Komplettes Redesign auf ein Material-3-artiges, vollständig selbst gehostetes Design-System (eigene Fonts/Icons, keine CDN-Abhängigkeit)
- Neue Top-Navigation (Desktop) und Bottom-Navigation (Mobil)
- Fix: mobile Navigation war unerreichbar, wenn EchoLox in LoxBerrys eigenem iframe-Wrapper angezeigt wird
- Echtes EchoLox-Logo im Header statt Platzhalter

**Alle drei Architekturen (arm64, armv7, amd64) lokal gebaut und getestet.**

## [v0.3.1-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.1-rc1) — 2026-08-08 — Prerelease

Build aus lokal kompilierten Bins

**Commit:** chore: bump to 0.3.1-rc1 — 0.3.0-rc1 was unreachable via SemVer

## [v0.3.0-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.0-rc1) — 2026-08-08 — Prerelease

Build aus lokal kompilierten Bins

**Commit:** chore: bump to 0.3.0-rc1 to publish a build with the mobile-nav fix

## [v0.3.0](https://github.com/BattloXX/EchoLox/releases/tag/v0.3.0) — 2026-08-07 — Prerelease

Build aus lokal kompilierten Bins

**Commit:** v0.3.0: LoxBerry 4 support, IPv6, reverse-proxy self-repair, UI redesign

## [v0.2.10-rc5](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10-rc5) — 2026-05-20 — Prerelease

### RC5 – Discovery nach echolox_* Namen (typunabhängig)

#### Problem (RC3/RC4)
LoxAPP3.json enthält keine Loxone-Peripherals (Virtuelle Eingänge). Die bisherige Suche nach `VirtualInput`-Typ fand daher 0 VIs.

#### Fix
Discovery sucht jetzt **nach Name** statt nur nach Typ:
- Jede Steuerung in LoxAPP3.json deren Name mit `echolox_` beginnt (Groß/Kleinschreibung egal) wird als VI erkannt — **unabhängig vom Loxone-Typ** (Switch, Pushbutton, VirtualInput, …)
- Zusätzlich bleibt die Typ-basierte Suche nach `VirtualInput`/`VirtualInputText` erhalten
- Direkte Steuerungen (Dimmer/Switch) die `echolox_`-Namen tragen werden nicht doppelt gezählt

#### Neues Diagnose-Logging
Im Log erscheint jetzt explizit:
```
viprobe: echolox_* controls in LoxAPP3.json: [echolox_licht_off echolox_licht_on …]
```
oder
```
viprobe: no echolox_* named controls in LoxAPP3.json — VIs are likely Loxone peripherals without visualization page
```

#### Entfernt
- Manuelle VI-Eingabe (war RC4, macht keinen Sinn)

#### Voraussetzung für echolox_* Discovery
Die Virtuellen Eingänge müssen in LoxAPP3.json sichtbar sein. Dazu muss in Loxone Config mindestens eine Visualisierungsseite für die VIs angelegt sein (rechtsklick → Zur Seite hinzufügen). Alternativ erscheinen sie wenn ein Programm-Block (Switch/Pushbutton) den echolox_-Namen trägt.

## [v0.2.10-rc4](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10-rc4) — 2026-05-20 — Prerelease

### Neu in RC4

#### Manuelle Virtual Input Eingabe
Da LoxAPP3.json keine Virtuellen Eingänge enthält (keine Loxone-Visualisierungsseite nötig), können `echolox_*`-VIs jetzt auf der **Discovery-Seite** direkt manuell eingetragen werden:
- Basis-Name eingeben (z.B. `echolox_wohnzimmer_licht`)
- Typ wählen (Switch / Dimmer / Color / Szene)
- Modus wählen (Impuls: _on/_off VIs · Ein/Aus: ein VI mit 1/0)
- Live-Vorschau der angelegten VI-Namen
- Gerät mit einem Klick anlegen

#### Außerdem (aus RC3)
- Automatische Erkennung von Dimmer, Switch, LightControllerV2 direkt aus LoxAPP3.json
- Verbesserte Fehlerbehandlung für Loxone LL-Error-Wrapper
- URL-Encoding für Steuerungsnamen mit Leerzeichen

## [v0.2.10-rc3](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10-rc3) — 2026-05-20 — Prerelease

### RC3 — Robuste Loxone-Verbindung & UI-Fix

#### Behobene Probleme

**Virtuelle Eingänge werden nicht gefunden**
Der Miniserver gibt bei Authentifizierungsfehlern HTTP 200 mit `{"LL":{"Code":"401",...}}` zurück statt HTTP 401. Bisher wurde dieser Fall nicht erkannt und die Suche lieferte still ein leeres Ergebnis. RC3 erkennt diesen Wrapper und gibt eine klare Fehlermeldung aus.

Zusätzlich: Loxone-Firmware verwendet `"Controls"` (Großbuchstabe) als JSON-Schlüssel — die Suche ist jetzt case-insensitiv und findet den Schlüssel unabhängig von der Schreibweise.

**Discovery UI: bereits importierte Geräte**
Bereits importierte VIs werden weiterhin angezeigt (zur Übersicht), haben aber keine Checkbox mehr — ein erneuter Import wird nicht angeboten.

#### Änderungen
- `FetchVIProposals`: LL-Wrapper-Erkennung + case-insensitive Controls-Suche
- `RefreshCache`: gleiche Fixes für die Status-Seite
- UI: keine Import-Checkbox für bereits vorhandene Geräte
- 2 neue Tests: `LLErrorWrapper`, `PascalCaseControls`

## [v0.2.10-rc2](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10-rc2) — 2026-05-20 — Prerelease

### Pre-release v0.2.10-rc2

Follows rc1. Fixes the remaining reason the Miniserver Discovery showed
\"keine Virtual Inputs gefunden\" even when echolox_* VIs existed.

#### Bug Fixes

**Case-insensitive VI detection** (`internal/loxone/viprobe.go`)

The prefix scan used a case-sensitive `strings.HasPrefix(name, "echolox_")`.
VIs named `EchoLox_*`, `ECHOLOX_*`, or any other mixed-capitalisation were
silently skipped — the scan returned zero proposals even though the VIs
existed on the Miniserver.

Fixed: `collectEcholoxVIs` now matches the prefix case-insensitively while
preserving the original VI name (required for HTTP commands to the Miniserver).
`viBaseToDisplayName` likewise strips the prefix case-insensitively.
New test file `viprobe_test.go` covers lowercase, mixed-case, nested
subcontrol, and display-name round-trip scenarios.

**Discovery UI shows all found VIs** (`webembed/web/assets/app.js`)

Previously, already-imported proposals were filtered out entirely and the page
showed the same generic message regardless of whether the Miniserver had zero
echolox_* VIs, or every VI was already imported. Both looked identical.

Fixed: All proposals are now rendered in the table. Already-imported entries
appear greyed-out with a ✓ badge and unchecked checkbox (user can still
re-import if needed). Two distinct info messages now explain each case:
- *No echolox_* VIs on Miniserver at all* → hint to check the VI prefix.
- *All VIs already imported* → overview of what's there, table still shown.

#### Also in this release (from rc1)

- **Bug 1 (rc1):** `handleDiscoverLoxoneImport` now explicitly maps discovered
  VI names to roles instead of relying on name-based auto-generation.
- **Bug 2 (rc1):** `selfHeal` no longer overwrites non-empty `VirtualInputs`
  on restart.

## [v0.2.10-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10-rc1) — 2026-05-20 — Prerelease

### Pre-release v0.2.10-rc1

This pre-release fixes two bugs in the Loxone Miniserver Discovery feature
that prevented Virtual Inputs from being correctly linked to imported devices.

#### Bug Fixes

**Bug 1 – VirtualInputs not populated during VI import**
(`internal/api/discover.go`)

When importing devices via the Discovery page, the `VirtualInputs` map was
never set. The code silently fell back to name-based auto-generation inside
`manager.Create()`, which breaks for any VI whose name contains characters
that don't survive the round-trip through `viBaseToDisplayName` → `NormalizeName`
(e.g. hyphens, multiple consecutive underscores). The import appeared to
succeed, but the device had no functional connection to the Miniserver —
Alexa commands were silently dropped.

**Fix:** `handleDiscoverLoxoneImport` now builds the `VirtualInputs` map
directly from the discovered VI names and their role suffixes, preserving
the exact names from the Miniserver.

---

**Bug 2 – selfHeal overwrites custom VirtualInputs on every restart**
(`internal/device/manager.go`)

`selfHeal()` regenerated the `VirtualInputs` map whenever it didn't match
the auto-generated pattern (derived from the device display name). This
silently destroyed any explicitly-set VI mappings — including those written
by the Bug 1 fix — each time EchoLox restarted. The dead helper
`virtualInputsMatch` is removed.

**Fix:** `selfHeal` now only generates `VirtualInputs` when the map is
completely absent (empty). Non-empty maps are trusted and left unchanged,
so discovery-imported or manually customised VI names survive restarts.

---

#### Upgrade Notes
Devices that were previously imported via Discovery and have empty or
incorrect `VirtualInputs` will **not** be auto-corrected by this release
(selfHeal no longer overwrites non-empty maps). To fix affected devices,
delete them and re-import via the Discovery page — they will now be created
with the correct VI names.

## [v0.2.10](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.10) — 2026-05-20

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- chore: update prerelease.cfg to v0.2.9-rc2 [skip ci]
- fix: populate VirtualInputs from discovered Miniserver VIs during import
- chore: update prerelease.cfg to v0.2.10-rc1 [skip ci]
- chore: bump version to 0.2.10, rebuild binaries [skip ci]
- fix: case-insensitive VI discovery + show already-imported proposals in UI

## [v0.2.9-rc2](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.9-rc2) — 2026-05-20 — Prerelease

### EchoLox v0.2.9-rc2 — Pre-release

#### Bugfix gegenüber rc1

**Fix: Backup-Pfad korrigiert — devices.json überlebt Plugin-Updates jetzt wirklich**

rc1 hatte den Backup-Pfad von `/tmp` nach `$LBPCFGDIR` verschoben. Das war falsch: LoxBerry's `purge_installation()` löscht **sowohl** `$LBPDATADIR` (das Original) **als auch** `$LBPCFGDIR` (das neue Backup) zwischen `preroot.sh` und `postroot.sh` — das Backup verschwand also genau dann, wenn es gebraucht wurde.

Das Backup liegt jetzt in `/var/tmp/EchoLox_devices.bak`:
- Liegt außerhalb des Plugin-Verzeichnisbaums → wird von `purge_installation` nicht angefasst
- Überlebt Neustarts (im Gegensatz zu `/tmp`, das auf LoxBerry ein tmpfs ist)

---

#### Alle Änderungen in v0.2.9 (gegenüber v0.2.8)

**Bug 1: devices.json überlebt Plugin-Updates** (`preroot.sh`, `postroot.sh`)

Backup/Restore über `/var/tmp/EchoLox_devices.bak` — außerhalb aller Plugin-Verzeichnisse, persistent über Reboots.

**Bug 2: Hue Dimmer Helligkeit funktioniert mit Loxone** (`internal/hue/api.go`)

Loxone verlangt, dass der `on`-Zustand vor dem Helligkeitswert gesendet wird. Bei reinen Brightness-Befehlen (ohne explizites `on`-Feld) wurde für `TypeDimmer`-Geräte kein `on=1` gesendet — Helligkeitsänderungen wurden von Loxone stillschweigend ignoriert. Behoben.

---

## [v0.2.9-rc1](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.9-rc1) — 2026-05-20 — Prerelease

### EchoLox v0.2.9-rc1 — Pre-release

#### Bug Fixes

**Fix: Device backup survives plugin updates** ([preroot.sh](preroot.sh), [postroot.sh](postroot.sh))

The `devices.json` backup was previously stored in `/tmp`, which is a volatile tmpfs on LoxBerry (Raspberry Pi). If the system lost power or `/tmp` was cleared between the `preroot.sh` and `postroot.sh` hooks during a plugin update, the entire device list was permanently lost.

The backup is now stored in `$LBPCFGDIR` (the plugin config directory, `/opt/loxberry/config/plugins/EchoLox/`), which LoxBerry preserves across plugin installs and updates.

**Fix: Hue Dimmer brightness now works correctly with Loxone** ([internal/hue/api.go](internal/hue/api.go))

Loxone requires the `on` state to be received on the virtual input before it processes a brightness value. When Alexa sent only a brightness command (without an explicit `on` field), no `on=1` was ever sent, causing brightness changes to be silently ignored.

For `TypeDimmer` devices, EchoLox now automatically sends `on=1` to the "on" virtual input before sending the brightness value whenever the request does not already include an explicit on/off state.

---

## [v0.2.9](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.9) — 2026-05-20

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- chore: update prerelease.cfg to v0.2.8 [skip ci]
- fix: persist devices.json backup in config dir across plugin updates
- fix: send on=1 before brightness for Hue Dimmer devices
- chore: bump version to 0.2.9, rebuild binaries [skip ci]
- chore: update prerelease.cfg to v0.2.9-rc1 [skip ci]
- fix: use /var/tmp for devices.json backup across plugin updates

## [v0.2.8](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.8) — 2026-05-18

### What's new

#### Geräteübersicht
- **Textfilter**: Suche in Echtzeit nach Name, Typ, Transport und Virtual Input Namen
- **Sortierung**: Klick auf Spaltenheader (Name, Typ, Modus, Transport) sortiert auf- oder absteigend — aktive Spalte wird grün hervorgehoben

#### Virtual Input Statusübersicht
- **Sortierung**: Alle 5 Spalten sortierbar (Status, Virtual Input Name, Gerät, Letzter Wert, Zuletzt gesendet)
- Kombiniert sich mit den bestehenden Status-Filterbuttons (OK / Not Found / Not Sent) und dem Suchfeld

#### Sonstiges
- Keine Backend-Änderungen, reines Frontend-Update

## [v0.2.7](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.7) — 2026-05-18

Build aus lokal kompilierten Bins

**Commit:** Merge pull request #1 from BattloXX/fix/loxberry-integration-and-bugs

## [v0.2.6](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.6) — 2026-05-17

Build aus lokal kompilierten Bins

**Commit:** v0.2.6: proper LoxBerry wrapper (loxberry_web.php) + wiki docs

## [v0.2.5](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.5) — 2026-05-17

Build aus lokal kompilierten Bins

**Commit:** v0.2.5: remove reverse proxy docs, fix LB UI link, add log rotation

## [v0.2.4](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.4) — 2026-05-17

Build aus lokal kompilierten Bins

**Commit:** chore: bump version to 0.2.4

## [v0.2.3](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.3) — 2026-05-12

Build aus lokal kompilierten Bins

**Commit:** chore: bump version to 0.2.3

## [v0.2.2](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.2) — 2026-05-10

Build aus lokal kompilierten Bins

**Commit:** feat: UUID salt via upnp.uuid config field, bump to v0.2.2

## [v0.2.1](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.1) — 2026-05-10

Build aus lokal kompilierten Bins

**Commit:** feat: add HTTP request logging middleware, bump to v0.2.1

## [v0.2.0](https://github.com/BattloXX/EchoLox/releases/tag/v0.2.0) — 2026-05-10

Build aus lokal kompilierten Bins

**Commit:** fix: executable bit auf Binaries setzen (Windows-Cross-Compile verliert +x)

## [v0.1.25](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.25) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** fix: postroot.sh auf neue Port-80-Architektur umgestellt, postinstall.sh vereinfacht

## [v0.1.24](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.24) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** feat: echte Versionsnummer in About-Seite und Startup-Log

## [v0.1.23](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.23) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** fix: version in binary verankern + Config-Migration auf Port 80

## [v0.1.22](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.22) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** fix: aarch64-Support und Binary-Auswahl bei Installation

## [v0.1.21](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.21) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** docs: README auf neue Port-80-Architektur umgeschrieben, wiki_template entfernt

## [v0.1.20](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.20) — 2026-05-09

Automatischer Build von \`main\`

**Commit:** feat: EchoLox exklusiv auf Port 80 + SSDP ohne Reverse-Proxy

## [v0.1.19](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.19) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** fix: postinstall.sh wieder auf Stub zurueckgesetzt

## [v0.1.18](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.18) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** docs: README und postinstall.sh aktualisiert

## [v0.1.17](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.17) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** feat: Loxone VI Discovery + Alexa-Geräteerkennung

## [v0.1.16](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.16) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** upnp: simplify multicast socket — no LoxBerry ssdpd conflict

## [v0.1.15](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.15) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** upnp: fix multicast socket to receive M-SEARCH from Alexa

## [v0.1.14](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.14) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** fix: SSDP und API exakt nach Home Assistant emulated_hue modelliert

## [v0.1.13](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.13) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** fix: SSDP/API-Version auf diyHue-kompatible Werte korrigiert

## [v0.1.12](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.12) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** fix: SSDP sendet jetzt alle 3 Service-Typen für Alexa-Discovery

## [v0.1.11](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.11) — 2026-05-08

Automatischer Build von \`main\`

**Commit:** fix: VI-Anzeige, Neustart, UI-Pfad /echoloxui/, stabile ECHO-Suche

## [v0.1.10](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.10) — 2026-05-07

Automatischer Build von \`main\`

**Commit:** feat: konfigurierbarer Schaltmodus pro Gerät (Ein/Aus oder Impuls)

## [v0.1.9](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.9) — 2026-05-07

Automatischer Build von \`main\`

**Commit:** feat: MQTT username/password + subscriptions; revamp README

## [v0.1.8](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.8) — 2026-05-07

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- docs: update README to reflect current state

## [v0.1.7](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.7) — 2026-05-07

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:


## [v0.1.6](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.6) — 2026-05-07

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- fix: devices survive updates, LoxBerry autoupdate, proper releases
- ci: add workflow_dispatch with prerelease option
- ci: fix YAML parse error in release.cfg update step

## [v0.1.5](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.5) — 2026-05-07

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- fix: backup JSON errors, config overwrite on update, directory permissions

## [v0.1.4](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.4) — 2026-05-07

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- feat: logging system, Alexa port-80 discovery fix, discovery_port config
- fix: switch proxy from nginx to Apache2 (LoxBerry uses Apache2)
- Fix SSDP discovery, settings overwrite, and Apache/LoxBerry conflicts
- fix: replace syscall.SetsockoptIpmreq with Syscall6 (unavailable in Go 1.22)
- fix: use hardcoded SO_REUSEPORT constant for ARM cross-compilation

## [v0.1.3](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.3) — 2026-05-06

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- Add restart, backup/restore + fix LoxBerry icons (v0.1.1)

## [v0.1.2](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.2) — 2026-05-06

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- Remove leftover files and add binaries to repo

## [v0.1.1](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.1) — 2026-05-06

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- Fix CI build: correct ldflags syntax for go build

## [v0.1.0](https://github.com/BattloXX/EchoLox/releases/tag/v0.1.0) — 2026-05-06

Git-Tag ohne GitHub-Release. Änderungen aus der Git-Historie:

- Initial EchoLox Go implementation
- Add cross-compiled binaries for arm64, armv7 and amd64
- Fix SSDP discovery and Hue API for Alexa compatibility
- Fix LoxBerry plugin installation errors
- Fix PHP 500 error in LoxBerry web frontend
- fix(webfrontend): remove loxberry_web.php dependency to fix HTTP 500
- chore: change default port from 8083 to 8079
- fix(postroot): install daemon init script during plugin setup
- feat: LoxBerry design, Miniserver test button, settings API, SSDP docs
- feat: separate on/off virtual inputs per device
- chore: rename VI prefix from ha_ to echolox_
- feat: About page, mobile nav, Miniserver config fix
- feat: icon, autostart via systemd, clean README
- fix(postroot): use LoxBerry plugin variables throughout
- Stop service before install/update and fix plugin icon
- Fix plugin icon installation for LoxBerry
- Fix miniserver reading, config persistence and add CI releases
