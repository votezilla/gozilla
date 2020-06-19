FROM golang:1.14.4-alpine
#FROM scratch

#COPY *.go /go/
#COPY common_passwords.txt /go/
#COPY static/*.png,static/*.jpg,static/*.ico,static/*.css,static/*.gif, /go/
#COPY static/newsSourceIcons/* /go/
#COPY ["static/votezilla logo/*", "/go/"]
#COPY static/templates/* /go/

COPY * /go/
COPY static/* /go/static/
COPY templates/* /go/templates/

RUN mkdir -p static/thumbnails

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

# Fetch all go library dependencies.
RUN go get -d -v

# We run go build to compile the binary executable of our Go program
RUN go build -o gozilla .

ENV PORT 8080
CMD ["./gozilla"]


