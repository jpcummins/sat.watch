# sat.watch

**sat.watch** is a Go application that continuously monitors the Bitcoin blockchain for activity (funds sent or received) on user-specified addresses and will send a notification if a transaction is detected.

This is the open-source, self-hosted version of sat.watch. If you prefer a hosted solution with additional features, a commercial version is available at [https://sat.watch](https://sat.watch).

![sat.watch app dashboard](https://sat.watch/static/screenshots/1.1.5-5.png "sat.watch app dashboard")

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Security

If you discover a security vulnerability within sat.watch, please send an email to the address listed at [https://sat.watch/contact](https://sat.watch/contact). You can find my PGP key there as well for secure communication. All security vulnerabilities will be promptly addressed.

## Dependencies


- [Bitcoin Core](https://github.com/bitcoin/bitcoin)
- Electrum server, such as [electrs](https://github.com/romanz/electrs) or [Fulcrum](https://github.com/cculianu/Fulcrum).
- Postgres
- SMTP credentials




## Installation

The recommended way to run sat.watch is using Docker, which provides a complete environment including Bitcoin Core and an Electrum server. For detailed Docker setup instructions, see the [Docker README](docker/README.md).

Alternative manual installation steps:


```bash
git clone https://github.com/jpcummins/satwatch
cd satwatch
go build .
go build -o user-create cmd/user/create/main.go
```



## Configuration

sat.watch is configured using environment variables. You can find a minimal example in [docker/.env.example](docker/.env.example). For enhanced security, it's recommended to encrypt your environment variables using [dotenvx](https://dotenvx.com/encryption). Here's an overview of the available configuration options:

Core Settings
- `ENVIRONMENT`: Set to "development" (default) or "production"
- `DATABASE_URL`: PostgreSQL connection string
- `URL`: Base URL used for links in email notifications (e.g., 'http://localhost:8080' for development)

Electrum Server Settings
- `ELECTRUM_HOST`: Hostname of your Electrum server
- `ELECTRUM_PORT`: SSL port of your Electrum server (typically 50002)

Email Settings
- `SMTP_HOST`: SMTP server hostname
- `SMTP_PORT`: SMTP server port
- `SMTP_USER`: SMTP username
- `SMTP_PASSWORD`: SMTP password

Bitcoin Core Setting
- `RPCAUTH`: Generated auth string for Bitcoin Core RPC
- `RPCUSER`: Bitcoin Core RPC username
- `RPCPASSWORD`: Bitcoin Core RPC password

Security
- `SECRET`: Used to encrypt session cookies



## Usage

**Docker (Recommended)**:
The recommended way to run sat.watch is using Docker, which provides a complete environment including Bitcoin Core and an Electrum server. For detailed Docker setup instructions, see the [Docker README](docker/README.md).

**Manual Installation**:

**Create a new user account**:

```bash
./user-create -username your_username -password your_password
```

**Start the server**:

```bash
./satwatch
```

Browse to [http://localhost:8080](http://localhost:8080)
