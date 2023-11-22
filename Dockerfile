# Use a Golang image as the base
FROM golang:latest as builder

# Set the working directory
WORKDIR /app

# Copy the Go app source code to the container
COPY . .

# Compile the Go app
RUN go build -o webm2mp4

# Use a clean image as the final base
FROM ubuntu:latest

# Install necessary dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    golang

# Set the working directory
WORKDIR /app

# Copy the compiled Go app from the previous stage
COPY --from=builder /app/webm2mp4 /app/webm2mp4

# Set the entrypoint for the container
ENTRYPOINT ["/app/webm2mp4"]

