     ████ █   █ █   █ █   █ █████ █████ ████  ███  ███   ███  █           ████ █████ ████  █████  ███  █   █ 
   █      █ █  ██ ██ ██ ██ █       █   █   █  █  █     █   █ █          █       █   █   █ █     █   █ ██ ██  
   ███    █   █ █ █ █ █ █ ████    █   ████   █  █     █████ █     ████  ███    █   ████  ████  █████ █ █ █   
     █   █   █   █ █   █ █       █   █  █   █  █     █   █ █              █   █   █  █  █     █   █ █   █    
████    █   █   █ █   █ █████   █   █   █ ███  ███  █   █ █████      ████    █   █   █ █████ █   █ █   █     

**A blazing-fast, peer-to-peer file transfer TUI built in Go.**

Symmetrical Stream allows two machines to establish a direct, encrypted WebRTC tunnel using a simple 6-digit PIN. Wrapped in a beautiful terminal interface, it transfers files directly between peers with zero middlemen, utilizing strict sequence enforcement and SHA-256 verification to guarantee data integrity.

---

## Features

* **Direct P2P Transfer:** Files are streamed directly over WebRTC Data Channels. Your data never touches a middleman or relay server.
* **Terminal Native:** A responsive, 60 FPS terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
* **Guaranteed Integrity:** Features built-in file chunking with strict sequence enforcement and automatic SHA-256 hash verification upon completion.
* **Zero Configuration:** No IP addresses or port forwarding required. Just share a 6-digit code to initiate the WebRTC handshake.

---

## Installation

### Using Homebrew (macOS / Linux)
The easiest way to install and keep the application updated is via the custom tap:

    brew install PeterJohnBishop/tap/symmetrical-stream

### Building from Source
Ensure you have Go 1.21+ installed, then clone the repository and build:

    git clone https://github.com/peterjohnbishop/symmetrical-stream.git
    cd symmetrical-stream
    go build -o symmetrical-stream main.go

---

## Usage

Launch the interactive terminal UI:

    symmetrical-stream

**To Send a File:**
1. Select the **Sender** role using `h`/`l` or the `Space` bar.
2. Enter the absolute path to the file you want to send.
3. Share the generated **6-digit Sender ID** with the receiver.

**To Receive a File:**
1. Select the **Receiver** role.
2. Enter the path to your desired download directory.
3. Input the **6-digit Sender ID** provided by the sender.
4. Watch the progress bar as the file streams directly to your disk!

---

## Architecture & Tech Stack

Symmetrical Stream bridges high-throughput networking with a single-threaded UI render loop. 

* **Go (Golang):** The core engine powering the high-speed background data pump.
* **Pion WebRTC:** Handles NAT traversal, ICE candidate negotiation, and the encrypted peer-to-peer data channels.
* **Bubble Tea & Lipgloss:** Powers the highly interactive, stylized TUI. 
* **Signaling Server:** A lightweight WebSocket server used *only* to exchange the initial WebRTC SDP offers/answers via the 6-digit PIN.

### Self-Hosting the Signaling Server

By default, the compiled binary points to the production signaling server. If you want to host your own or run it locally for development, you can override the connection URL using the `HOST` environment variable:

    HOST="localhost:8080" symmetrical-stream

---

## License

Distributed under the MIT License. See `LICENSE` for more information.
