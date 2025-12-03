LLAMA_SERVER_DIR = ../../vendor/llama.cpp/tools/server/
SERVER_FILES = server-common server-context server-http server-models server-queue server-task server
HEADERS = $(addsuffix .h, $(filter-out server, $(SERVER_FILES)))
SOURCES = $(addsuffix .cpp, $(SERVER_FILES))

.PHONY: clean all

all: $(HEADERS) $(SOURCES)

%.h: $(LLAMA_SERVER_DIR)/%.h
	cp $< $@

%.cpp: $(LLAMA_SERVER_DIR)/%.cpp
	cp $< $@
	@if [ "$@" = "server-http.cpp" ]; then patch $@ < server-http.patch; fi

clean:
	rm -f $(HEADERS) $(SOURCES)
