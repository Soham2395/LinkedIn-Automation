# Testing the LinkedIn Automation Bot

This guide explains how to run the bot in a safe, controlled manner to verify functionality.

## Prerequisites

1.  **Go 1.21+** installed.
2.  **Google Chrome** installed (required for the browser automation).
3.  **LinkedIn Account** credentials.

## Setup

1.  **Set Environment Variables**
    The bot requires your LinkedIn credentials to log in.
    ```bash
    export LINKEDIN_USERNAME="your_email@example.com"
    export LINKEDIN_PASSWORD="your_password"
    ```

2.  **Build the Bot**
    ```bash
    go build -o linkedin-bot ./cmd/linkedin-bot
    ```

## Running the Test

Run the bot directly from the terminal. The browser will open in **visible mode** (not headless) so you can watch the actions.

```bash
./linkedin-bot
```

## What to Expect

1.  **Authentication**:
    - The bot will open LinkedIn.
    - It will try to restore a session from `data/cookies.json` (if it exists).
    - If not, it will log in using your credentials.
    - **Note**: If LinkedIn asks for a CAPTCHA or 2FA code, the bot will detect the "checkpoint" and exit safely. You may need to solve it manually in a normal browser first to establish trust.

2.  **Search**:
    - It will search for "Software Engineer" in "San Francisco Bay Area".
    - It will scroll through the first page of results.
    - It will extract profile URLs.

3.  **Profile Visit & Connect**:
    - It will pick **one** profile from the search results.
    - It will visit the profile page.
    - It will attempt to click "Connect".
    - **Safety**: It checks if you are already connected. If the "Connect" button is missing or requires a password/email, it will skip safely.

4.  **Follow-Ups**:
    - It will check your "Connections" page for newly accepted requests.
    - If it finds any *that the bot previously interacted with*, it will send a follow-up message.

## Safety Features Active

- **Risk Engine**: Tracks cumulative risk. If it exceeds the daily limit (100 for Normal profile), the bot stops.
- **Stealth Engine**: Adds human-like delays and mouse movements.
- **Imperfection Engine**: Occasionally simulates typos or mouse slips (you might see it backspace and retype).
- **State Persistence**: All progress is saved to `data/state.json`. If you restart, it remembers who it already contacted.

## Troubleshooting

- **"checkpoint detected"**: LinkedIn flagged the login. Log in manually in Chrome to clear the flag.
- **"connect button not found"**: The profile might be out of network (3rd degree) or have "Follow" as the primary action. The bot skips these safely.
- **Browser crashes**: Ensure Chrome is installed and not blocked by antivirus.
