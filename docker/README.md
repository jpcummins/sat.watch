# Docker Setup for sat.watch

This folder contains a sample `docker-compose.yml` that is designed to get you up and running quickly with sat.watch. The compose file builds multiple services, including an Electrum server (electrs) and a Bitcoin core node, to facilitate a working blockchain indexing environment.

## Prerequisites

Before you begin, please ensure that you have the following installed on your system:

- [Docker Engine](https://docs.docker.com/engine/overview/) – Provides the container runtime.
- [Docker Compose](https://docs.docker.com/compose/overview/) – Manages multi-container Docker applications.

Additionally, you will need a machine with sufficient disk space (at least 1-1.5 TB available) because the initial blockchain download is resource-intensive.

## Step 1. Configure the Environment

The repository includes an environment file template (.env.example) with default settings. Copy this file to create your own .env file and adjust the configurations as needed for your environment.

    cp .env.example .env

## Step 2. Prepare the Electrum Server

There isn't an official Docker image available for electrs at the moment. Therefore, this setup pulls the source code and builds it locally. Note that we're using an unofficial build of Bitcoin core in this setup. While building Bitcoin core from source is generally recommended for security and performance, it is beyond the scope of this quick setup guide.

Clone the electrs repository with the following command:

    git clone https://github.com/romanz/electrs.git

## Step 3. Build the Docker Services

With your environment configured and the source code prepared, the next step is to build the services defined in the docker-compose file. This command uses the variables you set in .env.

    docker compose --env-file .env build

## Step 4. Start the Services

After building the images successfully, you can start the services using Docker Compose in detached mode:

    docker compose --env-file .env up -d

What to Expect:
- Initial Blockchain Download: The blockchain will start downloading and syncing. This is a resource-intensive process and may take one to two days to complete, depending on your hardware and internet speed.
- Accessing sat.watch: Once the blockchain is fully synced and the indexing is complete, you should be able to access the sat.watch application by visiting http://localhost:8080.

## Creating Your First User Account

Before you can use sat.watch, you need to create a user account. With the Docker setup, you can do this using the included CLI tool:

```bash
docker compose exec satwatch /create-user -username your_username -password your_password
```

Note: Make sure your password is at least 6 characters long.

## Monitoring and Troubleshooting

Logs:
If you want to check the progress or debug any issues, you can view the logs by running:

    docker compose logs -f
