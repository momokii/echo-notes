# Echo-Notes Web App

This repository contains the source code for the **Echo-Notes Web App**. The primary goal of this application is to provide an intuitive platform for recording audio directly from web browsers, with AI-powered transcription and summarization capabilities. Echo-Notes helps users capture and extract key insights from meetings efficiently without the need for manual note-taking.

In addition to Echo-Notes, this project now includes **Single Sign-On (SSO)** integration for secure user authentication. The SSO implementation is based on [go-sso-web repository](https://github.com/momokii/go-sso-web), providing a seamless and secure authentication experience.

## **New Feature: Credit System Integration**
This project now includes a **credit system** integrated with the SSO authentication. Each feature that using LLM will consumes a different amount of credits, referred to as "feature cost," depending on the complexity of the processing required. The credit system ensures fair usage and allows users to manage their usage effectively.

## Features

- **Live Audio Recording**: Record meetings directly from your web browser with a simple, intuitive interface.
- **AI-Powered Transcription**: 
  - Utilizes OpenAI's Whisper model for accurate speech-to-text conversion.
  - Captures spoken content with high precision for further processing.
- **Dual Summarization Options**:
  - **TLDR Version**: Quick, concise summaries capturing the essential points.
  - **Comprehensive Summary**: Detailed overviews of the entire meeting content.
- **Clean User Interface**: 
  - Simple controls for recording, stopping, and processing audio.
  - Easy access to both transcripts and summaries.
- **Minimalist Frontend**: Built using **HTML**, **Bootstrap**, and **jQuery** for a clean and responsive design.
- **Backend Framework**: Powered by **Golang Fiber**, ensuring fast and efficient performance.

## Future Enhancements

- **Authentication System**: User accounts for secure access to recordings and summaries.
- **Database Integration**: Persistent storage for meeting recordings, transcripts, and summaries.
- **Credit System Integration**: Potential implementation of a token system for LLM usage.
- **Multi-App Integration**: Possibility to expand into a centralized platform for various productivity tools.

## Purpose

This project was developed as part of my portfolio to showcase practical implementation of AI-powered speech processing. Echo-Notes demonstrates the integration of modern web technologies with state-of-the-art AI models to create a useful productivity tool that can help professionals make the most of their meeting time.

## Tech Stack

- **Backend**: Golang Fiber framework
- **Frontend**: HTML, Bootstrap, and jQuery
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
