###### LEVEl I ####################
# ARGS
ARG BINARY=data-sync-agent
ARG PLATFORM=linux
ARG GOARCH=amd64
ARG DESTINATION=/app/bin-${PLATFORM}-${GOARCH}
# First stage: build the executable.
# Pull base image from local cerebrum microservice registry.
FROM registry.gitlab.com/cerebrum-dockerbase/microservice-dockerbase:latest as builder
# Renew the global variables, to be used after from
ARG PLATFORM

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /app
COPY . /root
# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./go.mod ./go.sum ./
# update the go mod depen. based on branch
RUN /root/app.sh
# we are replacing mod above, which will works fine with vendor, so please dont use download, which will download the dependencies replaced also...so latest will be taken from sum file, which leads to inconsistency..
RUN go mod download

# Copy the code from the context(root).
COPY . .

# building the docker image based on os
RUN make ${PLATFORM}

######### LEVEL II ##########################
# To minimize the image size, we are rebuilding from scratch base img.
FROM scratch

# Renew the global variables, to be used after from
ARG BINARY
ARG PLATFORM
ARG GOARCH
ARG DESTINATION

# Import the user and group files from the first stage.
COPY --from=builder /user/group /user/passwd /etc/
# copying all the required files from builder base img to scratch img..
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copying compiled binary from Ist stage (builder) to current container...
COPY --from=builder /app/${BINARY}-${PLATFORM}-${GOARCH} /app/bin-linux-amd64 

# Perform any further action as an unprivileged user.
USER nobody:nobody

# Running the App, build from scratch, so we are using CMD...
CMD [ "/app/bin-linux-amd64" ]
