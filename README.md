# LLM Search Assistant

This project is an interactive terminal-based assistant that combines the power of Large Language Models (LLMs) and a search tool for DuckDuckGo. It retrieves search results based on user input and generates contextually relevant responses using an LLM.

## Features

- LLM-Driven Responses: The assistant uses an LLM to provide answers based on search results and conversation history.
- Search Integration: Searches the web via DuckDuckGo for additional information and relevant links.
- Interactive CLI: The program is fully interactive, allowing users to ask questions in a conversation loop.
- Streaming and Timeout Support: Responses are streamed and support timeouts for long-running operations.

## Usage

1. Run the program:

   ```bash
   ./search-assistant
   ```

2. Enter questions in the terminal. The assistant will provide answers based on search results and LLM-generated responses.
3. To quit, type `quit` at any prompt.

## Example
``` bash
Enter your question (or 'quit' to exit): What is the capital of France?
Generating response...
AI: The capital of France is Paris.
```

## Configuration

The program uses the following configurable settings:

- Model: The LLM model to use (configured in ModelConfig).
- Timeout: The maximum duration (in seconds) to wait for the LLM to respond.

By default, it uses the llama3.1 model and a timeout of 2 minutes.

## Error Handling

- CAPTCHA Detection: If a CAPTCHA is detected during the DuckDuckGo search, the program will notify the user and advise retrying later.
- Timeouts: If the LLM call takes too long, the program gracefully handles timeouts and notifies the user.