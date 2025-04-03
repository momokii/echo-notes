# Echo-Notes Web App

This repository contains the source code for the **Echo-Notes Web App**. The primary goal of this application is to provide an intuitive platform for recording audio directly from web browsers, with AI-powered transcription and summarization capabilities. Echo-Notes helps users capture and extract key insights from meetings efficiently without manual note-taking.

In addition to core functionalities, this project now integrates **Single Sign-On (SSO)** for secure user authentication, based on the [go-sso-web repository](https://github.com/momokii/go-sso-web), ensuring a seamless and secure login experience.

## Key Features

- **Live Audio Recording**  
  Record meetings directly from your web browser with a simple, intuitive interface.

- **AI-Powered Transcription**  
  - Utilizes OpenAI's Whisper model for accurate speech-to-text conversion.
  - Captures spoken content precisely for further processing.

- **Dual Summarization Options**  
  - **TLDR Version**: Quick, concise summaries capturing the essential points.
  - **Comprehensive Summary**: Detailed overviews of the entire meeting content.

- **Recording and Saving Summaries**  
  - Automatically saves the generated meeting summaries.
  - Access and manage summaries via a dedicated dashboard.

- **Summary Management Dashboard**  
  - **Summary Overview**: A new dashboard displaying all saved meeting summaries.
  - **Edit and Delete Options**: Users can modify the name and description (while keeping the summary content immutable) or delete unwanted summary data.

- **Advanced AI Feature in Dashboard**  
  - Users can select up to 5 summary entries.
  - The system will generate a combined summary divided into:
    - **Overview**: A unified summary of all selected data.
    - **Meeting Summaries**: Individual summaries for each selected meeting.
    - **Next Steps**: Recommendations and actionable next steps derived from all selected data.

- **Clean User Interface**  
  - Simple controls for recording, stopping, and processing audio.
  - Easy access to transcripts and summaries through a clean and modern UI.

- **Minimalist Frontend**  
  Built using **HTML**, **Bootstrap**, and **jQuery** for a responsive and modern design.

- **Backend Framework**  
  Powered by **Golang Fiber** for fast and efficient performance.

## Purpose

This project was developed as part of my portfolio to demonstrate a practical implementation of AI-powered speech processing. Echo-Notes showcases the integration of modern web technologies with state-of-the-art AI models to create a powerful productivity tool that helps professionals maximize their meeting efficiency.

## Tech Stack

- **Backend**: Golang Fiber framework
- **Frontend**: HTML, Bootstrap, and jQuery
- **Database**: Postgresql
- **AI Models**: 
  - OpenAI Whisper for speech-to-text transcription
  - OpenAI GPT series for text summarization

## Development Status

Echo-Notes is currently in active development with core recording and summarization features implemented. Authentication and database integrations are planned for future releases.


## How to Run

1. **Clone the repository:**
   ```bash
   git clone github.com/momokii/echo-notes
   ```

2. **Install Go:**
   - Make sure Go is installed. You can check by running:
    ```bash
     go version
     ```

3. **Run the server:**
     - Start the server using the following command:
     ```bash
     go run main.go
     ```

4. **Optional: Use Air for Hot Reloading:**
     - If you want hot reloading during development, you can use [Air](https://github.com/cosmtrek/air).
   - Start the server with Air by running:
    ```bash
     air
     ```

5. **Access the website:**
     - Open your browser and go to `http://localhost:3003` (or the specified port).
