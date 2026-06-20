# syntax=docker/dockerfile:1
# Full multi-stage build for local dev, CI, and `docker build -f Dockerfile`.
FROM golang:1.26.4-alpine3.22 AS builder

RUN apk add --no-cache git ca-certificates

RUN go install github.com/a-h/templ/cmd/templ@v0.3.977

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate

ARG VERSION=dev
ARG COMMIT=none
ARG BUILDDATE=unknown
ARG BRANCH=unknown

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w \
	-X github.com/hrodrig/kui/internal/version.Version=${VERSION} \
	-X github.com/hrodrig/kui/internal/version.Commit=${COMMIT} \
	-X github.com/hrodrig/kui/internal/version.BuildDate=${BUILDDATE} \
	-X github.com/hrodrig/kui/internal/version.Branch=${BRANCH}" \
	-o /usr/local/bin/kui ./cmd/kui

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /usr/local/bin/kui /usr/local/bin/kui

USER nonroot:nonroot

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/kui"]
CMD ["serve"]
