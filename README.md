# Video Clipper

A containerized web app for clipping online videos from [over 1000 sites](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md). 

Simply provide a URL, define your start and end times, choose the video quality, and download the perfect segment.

## Branches

This project has two main branches, each implementing a distinct real-time update strategy:

*   `sse-version` (this branch): Uses Server-Sent Events (SSE).

*   `main`: Uses WebSockets.


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