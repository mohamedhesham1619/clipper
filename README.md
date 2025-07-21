# Video Clipper

A containerized web app for clipping online videos from [over 1000 sites](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md). 

Simply provide a URL, define your start and end times, choose the video quality, and download the perfect segment.

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



## 🚀 Live Demo

Try the hosted app with extra features at: [https://videoclipper.online](https://videoclipper.online)

**The hosted app includes:**
- SEO-optimized pages for better discoverability
- Enhanced real-time progress updates
- Rate limiting to prevent abuse
- Contact form for user feedback
- All the core features of this open-source version

---


