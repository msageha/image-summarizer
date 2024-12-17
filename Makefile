DIR ?= ./images
OUT ?= output.png
N ?= 3
TILE ?= 300

TARGET = image-summarizer
SOURCES = main.go
GOFLAGS =

.PHONY: all build run tidy clean

all: build

build:
	@echo "Building the binary..."
	go build $(GOFLAGS) -o $(TARGET) $(SOURCES)

run: build
	@echo "Running the image summarizer generator..."
	./$(TARGET) -dir $(DIR) -out $(OUT) -n $(N) -tile $(TILE)

tidy:
	@echo "Running go mod tidy..."
	go mod tidy

clean:
	@echo "Removing binary..."
	rm -f $(TARGET)
