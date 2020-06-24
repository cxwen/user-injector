# Build the manager binary
FROM xwcheng/kubebuilder:2.3.1 as builder

WORKDIR /workspace

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod vendor -a -installsuffix cgo -o user-injector

FROM alpine:3.11.2
WORKDIR /
COPY --from=builder /workspace/user-injector .

ENTRYPOINT ["/user-injector"]
