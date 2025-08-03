# Video Clipper

A containerized web app for clipping online videos from [over 1000 sites](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md). 

Simply provide a URL, define your start and end times, choose the video quality, and download the perfect segment.

## Table of Contents
- [Tech Stack](#tech-stack)
- [Running with Docker](#running-with-docker)
- [Demo](#demo)
- [Hosted Version (Live Demo)](#hosted-version-live-demo)
  - [Key Improvements in the Hosted Version](#key-improvements-in-the-hosted-version)
  - [Hosted Tech Stack](#hosted-tech-stack)
  - [Internal State Management](#internal-state-management)
  - [Request Flow Diagram](#request-flow-diagram)

## Branches

This project has two main branches, each implementing a distinct real-time update strategy:

*   `main` (this branch): Uses Server-Sent Events (SSE).

*   `socket-version`: Uses WebSockets.

## Tech Stack

- **Backend:** Go
- **Frontend:** AI-Generated (HTML/CSS/JS)
- **Video Processing:** [yt-dlp](https://github.com/yt-dlp/yt-dlp), [ffmpeg](https://ffmpeg.org/)
- **Real-time Communication:** Server-Sent Events (SSE) 
- **Containerization:** Docker



## Running with Docker

1. **Build the Docker image:**
   ```sh
   docker build -t clipper .
   ```

2. **Run the Docker container:**
   ```sh
   docker run -p 8080:8080 clipper
   ```

3. **Access the application:**  
   Open [http://localhost:8080](http://localhost:8080) in your browser.

## Demo

https://github.com/user-attachments/assets/631e8b5f-0b27-40ea-a1c6-a669859336ea



## Hosted Version (Live Demo)

You can try the hosted app at: https://videoclipper.online

### Key Improvements in the Hosted Version

- A cleaner and more responsive UI  
- Enhanced real-time progress updates  
- SEO-optimized pages for better discoverability  
- Rate limiter for the submit endpoint  
- Contact feature to receive user feedback  
- Same backend core and clipping functionality as this repo

---

### Hosted Tech Stack

- **UI Design:** Visily  
- **Frontend:** AI-generated HTML/CSS/JS (mostly using [Windsurf](https://windsurf.com/))  
- **Backend:** Go  
- **Video Processing:** yt-dlp, ffmpeg  
- **Real-Time Communication:** Server-Sent Events (SSE)  
- **Containerization:** Docker  
- **Hosting:** Google Cloud
- **Analytics:** Google Analytics

---

### Internal State Management

The backend uses mutex-protected maps to safely manage shared state across concurrent client sessions.

All shared data is encapsulated in a central struct guarded by a `sync.RWMutex`, ensuring thread-safe reads/writes and avoiding race conditions.


**Maps Used Internally:**

- `processID → filePath`  
  Stores the output file path. Used to serve the file to the client.

- `processID → runningProcess`  
  Tracks the `*exec.Cmd` instance running the `ffmpeg` download command. Used to cancel downloads if a client disconnects.

- `processID → progressChan`  
  A Go channel used to send real-time events (`title`, `progress`, `complete`, `error`) to the client via SSE.


---

### Request Flow Diagram
The following diagram illustrates the lifecycle from request submission to file download:


```mermaid
sequenceDiagram
  participant Client as Client
  participant Backend as Backend
  participant SharedStore as SharedStore

  %% Submit Phase
  Client ->> Backend: POST /submit (URL, start time, end time, quality)
  Backend -->> Backend: Start the download process
  Backend ->> SharedStore: Store processID → filePath
  Backend ->> SharedStore: Store processID → runningProcess
  Backend ->> SharedStore: Store processID → progressChan
  Backend -->> Client: { processID }

  %% Progress Phase (SSE stream)
  Client ->> Backend: GET /progress/{processID} (SSE)
  Backend ->> SharedStore: Read from progressChan

  %% SSE Events
  Backend -->> Client: SSE event: title
  Backend -->> Client: SSE event: progress (repeated)
  Backend -->> Client: SSE event: complete
  Backend -->> Client: SSE event: error (if any)

  %% Download Phase
  Client ->> Backend: GET /download/{processID}
  Backend ->> SharedStore: Retrieve filePath
  SharedStore ->> Backend: filePath
  Backend -->> Client: Serve file
Backend -->> SharedStore: Remove the file and all associated resources


