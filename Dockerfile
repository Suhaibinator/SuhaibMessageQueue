# Use a minimal base image
FROM alpine:latest

# Set the working directory in the container
WORKDIR /app

# Create a directory for the database
VOLUME /db

# Copy the binary to the working directory
COPY SuhaibMessageQueue .

# Make the binary executable
RUN chmod +x SuhaibMessageQueue

# Specify the command to run when the container starts
CMD [ "./SuhaibMessageQueue" ]