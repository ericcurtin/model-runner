cmake_minimum_required(VERSION 3.13)

project(
    com.docker.llama-server.native
    DESCRIPTION "DD inference server, based on llama.cpp native server"
    LANGUAGES C CXX
)

option(DDLLAMA_BUILD_SERVER "Build the DD llama.cpp server executable" ON)
option(DDLLAMA_BUILD_UTILS "Build utilities, e.g. nv-gpu-info" OFF)

set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

if (DDLLAMA_BUILD_SERVER)
    # Enable building the vanilla llama.cpp server
    set(LLAMA_BUILD_COMMON ON CACHE BOOL "" FORCE)
    set(LLAMA_BUILD_TOOLS ON CACHE BOOL "" FORCE)
    set(LLAMA_BUILD_SERVER ON CACHE BOOL "" FORCE)
    add_subdirectory(vendor/llama.cpp)

    # Create a custom target that copies/renames the server binary after build
    if (WIN32)
        set(SERVER_OUTPUT_NAME com.docker.llama-server.exe)
    else()
        set(SERVER_OUTPUT_NAME com.docker.llama-server)
    endif()

    add_custom_target(docker-llama-server ALL
        DEPENDS llama-server
        COMMAND ${CMAKE_COMMAND} -E copy
            $<TARGET_FILE:llama-server>
            ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/${SERVER_OUTPUT_NAME}
        COMMENT "Creating ${SERVER_OUTPUT_NAME} from llama-server"
    )
endif()

if (WIN32 AND DDLLAMA_BUILD_UTILS)
    add_subdirectory(src/nv-gpu-info)
endif()
