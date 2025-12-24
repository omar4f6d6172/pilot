# Pilot: Systemd Socket Activation Control-Plane CLI

## Exposé zur Bachelorarbeit

**Thema:** Ressourceneffiziente Bereitstellung von Webdiensten mit systemd Socket Activation und Entwicklung einer Control-Plane CLI
**Eingereicht von:** Omar Altanbakji (Matrikelnummer: 587682), Noah Eid (Matrikelnummer: 584965)
**Betreuer:** Prof. Dr.-Ing. Marcin Brzozowski
**Datum:** 15. November 2025

---

## 1. Problemstellung und Forschungsleitfrage

Die moderne Bereitstellung von Webdiensten, oft mittels Container-Virtualisierung (z. B. Docker, Podman), führt in Multi-Tenant-Umgebungen zu erheblichem Ressourcen-Overhead, insbesondere beim Arbeitsspeicher. Jeder Tenant benötigt isolierte Runtimes und Prozessgruppen, unabhängig von der tatsächlichen Aktivität.

**Pilot** bietet einen alternativen Architekturansatz mittels **systemd Socket Activation**. Dabei überwacht `systemd` zentrale Netzwerk-Sockets und startet den eigentlichen Dienst ("Lazy Loading") erst bei eingehenden Anfragen. Bei Inaktivität wird der Dienst beendet. In Kombination mit Kernel-nativen Sicherheitsfeatures (cgroups v2, User Namespaces) und PostgreSQL Peer-Authentication ermöglicht dies einen extrem ressourceneffizienten Betrieb.

Die größte Hürde für den produktiven Einsatz dieser Technologie ist die fehlende Orchestrierung. Es existiert kein standardisiertes Tooling, das die komplexe Konfiguration von Linux-Usern, systemd-Units, Datenbank-Rollen und Reverse-Proxy-Routen automatisiert. Diese manuelle Einrichtung ist fehleranfällig, zeitaufwendig und schlecht skalierbar.

**Forschungsleitfrage:**
Ist es möglich, durch die Entwicklung eines spezialisierten Golang-basierten Control-Plane CLI-Tools die Orchestrierung von systemd Socket Activation auf einem Linux-System so zu automatisieren, dass eine signifikante Ressourceneinsparung gegenüber permanent laufenden Containern erzielt wird, ohne die Administrationskomplexität zu erhöhen oder die Sicherheit zu kompromittieren?

---

## 2. Technologische Grundlagen

Pilot setzt bewusst auf leistungsfähige Linux-Bordmittel statt auf komplexe Container-Orchestrierer:

*   **systemd:** Prozessmanager und Socket-Listener. Nutzt Socket-, Service-Templates und `RuntimeMaxSec` für das Lifecycle-Management.
*   **Linux Kernel Isolation:** Dedizierte System-User und cgroups zur Isolierung der Tenants.
*   **PostgreSQL Peer-Authentication:** Authentifizierung über den Kernel (SO_PEERCRED), wodurch Passwörter in Konfigurationsdateien obsolet werden.
*   **Caddy Web Server:** Dient als Reverse Proxy, dynamisch konfiguriert über seine REST-API.

---

## 3. Features der Control Plane CLI

Pilot automatisiert folgende Prozesse:

*   **User & Environment Management:**
    *   Anlegen isolierter Linux-User für jeden neuen Service via Syscalls.
    *   Einrichten der Dateisystemberechtigungen nach dem Prinzip der minimalen Rechte (Least Privilege).
*   **systemd Orchestrierung:**
    *   Dynamische Generierung und Platzierung von `.socket` und `.service` Unit-Files im User-Scope.
    *   Interaktion mit `systemctl` zur Aktivierung und Überwachung der Sockets.
*   **Datenbank-Automation:**
    *   Verbindung zur lokalen PostgreSQL-Instanz.
    *   Erstellung von Datenbank-Usern, die 1:1 auf die System-User gemappt sind, sowie Initialisierung der Tenant-Datenbanken.
*   **Reverse Proxy Konfiguration:**
    *   Ansteuerung des Caddy-Servers über einen integrierten API-Client.
    *   Dynamisches Routing von Domains (z. B. `tenant1.example.com`) auf die entsprechenden lokalen Unix-Sockets von systemd.
*   **Logging & Observability:**
    *   Administrations-Aktionen werden protokolliert.
    *   Nutzung von `journalctl` zur Aggregation der Logs der gestarteten Sub-Prozesse.

---

## 4. Voraussetzungen

Um Pilot nutzen zu können, muss Ihr Linux-System (z.B. Fedora, Debian) die folgenden Komponenten bereitstellen:

*   **Go (Golang):** Version 1.25.4 oder neuer.
*   **systemd:** Ein Init-System mit Unterstützung für User-Units und `loginctl`.
*   **PostgreSQL:** Eine laufende PostgreSQL-Instanz. Die `pg_hba.conf` muss `peer` Authentifizierung für lokale Verbindungen unterstützen (Standardkonfiguration unter Linux ist oft ausreichend).
*   **Caddy:** Ein laufender Caddy Web Server, dessen Admin API auf `localhost:2019` erreichbar ist.
*   **Standard Linux Tools:** `useradd`, `loginctl`, `systemctl`, `createuser`, `createdb`.

---

## 5. Installation

1.  **Repository klonen:**
    ```bash
    git clone [repository-url]
    cd pilot
    ```

2.  **Abhängigkeiten installieren:**
    ```bash
    go mod tidy
    ```

3.  **CLI und Backend-Anwendung kompilieren:**
    ```bash
    go build -o bin/pilot main.go
    go build -o bin/user-rest-api test/user-rest-api.go
    ```

4.  **Backend-Anwendung installieren:**
    Die systemd-Units erwarten, dass die Beispiel-Backend-Anwendung (`user-rest-api`) unter `/usr/local/bin/` verfügbar ist.
    ```bash
    sudo cp bin/user-rest-api /usr/local/bin/
    ```

---

## 6. Nutzung

Das `pilot` CLI ist das zentrale Werkzeug zur Verwaltung Ihrer Tenants. **Alle Befehle, die Systemänderungen vornehmen (z.B. Benutzer erstellen, systemd-Units konfigurieren), müssen mit `sudo` ausgeführt werden.**

### Tenants vollständig provisionieren

Der `create-tenant` Befehl führt die komplette Orchestrierung für einen neuen Tenant durch: Linux-Benutzer, PostgreSQL-Datenbank, systemd-Units und Caddy-Proxy.

```bash
sudo ./bin/pilot create-tenant --name="mytenant" --domain="mytenant.localhost" --idle="5min"
```
*   `--name`: Der Name des Tenants (wird als Linux-Benutzername, PostgreSQL-Rolle und Datenbankname verwendet). **(Erforderlich)**
*   `--domain`: Die Domain, unter der der Dienst erreichbar sein wird. Wenn nicht angegeben, wird `[name].localhost` verwendet.
*   `--idle`: Die Zeitspanne, nach der der Dienst bei Inaktivität beendet wird (z.B. "10s", "1min", "1h").

### Einzelne Schritte manuell ausführen

Sie können die einzelnen Schritte der Orchestrierung auch separat ausführen:

*   **Linux-Benutzer erstellen:**
    ```bash
    sudo ./bin/pilot create-user --name="myuser"
    ```

*   **systemd-Units einrichten:**
    ```bash
    sudo ./bin/pilot setup-systemd --name="myuser" --idle="10s"
    ```

*   **Caddy-Proxy konfigurieren:**
    ```bash
    sudo ./bin/pilot setup-proxy --name="myuser" --domain="myuser.example.com"
    ```

### Dienststatus überprüfen

Überprüfen Sie den Status der systemd-Dienste und die zugehörigen Benutzer.

```bash
./bin/pilot check
# Oder spezifische Dienste:
./bin/pilot check caddy.service postgresql.service user@1000.service
```

### Fake-Benutzer für Tests erstellen

Zum Testen und Evaluieren können Sie mehrere Fake-Benutzer auf einmal erstellen:

```bash
sudo ./bin/pilot create-fake-users --count=5 --idle="10s"
```

---

## 7. Beispielanwendung (`user-rest-api`)

Die mitgelieferte Go-Anwendung `test/user-rest-api.go` dient als Minimalbeispiel für einen Webdienst. Sie läuft als der jeweilige Tenant-Benutzer unter systemd Socket Activation und demonstriert:

*   Den Zugriff auf Umgebungsvariablen (`PORT`).
*   Die Authentifizierung und Verbindung zur PostgreSQL-Datenbank über **Peer Authentication** (als der Benutzer selbst, ohne Passwort).

Nach erfolgreicher Provisionierung eines Tenants (z.B. `mytenant` mit `mytenant.localhost` als Domain), können Sie den Dienst im Browser oder mit `curl` aufrufen:

```bash
curl http://mytenant.localhost
```
Die Antwort sollte JSON enthalten, das den aktuellen Linux-Benutzer und den Status der PostgreSQL-Verbindung anzeigt (z.B. `"db_status": "Connected"`).

---

## 8. Projektstruktur

*   `main.go`: Einstiegspunkt der CLI-Anwendung.
*   `cmd/`: Enthält die Implementierung der Cobra-Befehle:
    *   `check.go`: Überprüft den Status von systemd-Diensten.
    *   `createFakeUsers.go`: Erstellt mehrere Test-Tenants.
    *   `createTenant.go`: Der Hauptbefehl zur vollständigen Tenant-Provisionierung.
    *   `createUser.go`: Erstellt isolierte Linux-Benutzer mit Lingering.
    *   `root.go`: Die Basis des Cobra-CLI.
    *   `setupDatabase.go`: Konfiguriert PostgreSQL-Benutzer und -Datenbanken.
    *   `setupProxy.go`: Konfiguriert den Caddy Reverse Proxy.
    *   `setupSystemd.go`: Generiert und installiert systemd User-Units für Socket Activation.
    *   `utils.go`: Hilfsfunktionen zum Ausführen von Befehlen als anderer Benutzer und Schreiben von Dateien.
*   `test/user-rest-api.go`: Die Beispiel-Backend-Anwendung, die von systemd gestartet wird.
*   `bin/`: Ausgabeverzeichnis für die kompilierten Binaries.

---

## 9. Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Details finden Sie in der `LICENSE`-Datei.

---

## 10. Mitwirkende

*   Omar Altanbakji
*   Noah Eid
*   Betreuer: Prof. Dr.-Ing. Marcin Brzozowski