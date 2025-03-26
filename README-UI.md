# Stathera UI

This is the frontend user interface for the Stathera monetary system. It provides a modern, responsive web interface for interacting with the Stathera API.

## Overview

The Stathera UI is built with Next.js, TypeScript, and Tailwind CSS. It provides a clean, intuitive interface for users to interact with the Stathera monetary system, including:

- Viewing system metrics and statistics
- Managing accounts
- Submitting and tracking transactions
- Monitoring the ledger, transaction, and settlement layers

## Architecture

The UI follows a modern frontend architecture:

1. **Next.js Framework**: Provides server-side rendering, static site generation, and API routes
2. **TypeScript**: Ensures type safety throughout the codebase
3. **Tailwind CSS**: Utility-first CSS framework for rapid UI development
4. **SWR**: React Hooks library for data fetching with caching and revalidation
5. **Axios**: HTTP client for API communication

## Key Features

- **Dashboard**: Overview of system metrics, account balances, and recent transactions
- **Account Management**: Create, view, and manage accounts
- **Transaction Processing**: Submit transactions and view transaction history
- **System Information**: Monitor the ledger, transaction engine, and settlement process
- **Responsive Design**: Works on desktop and mobile devices

## Getting Started

### Prerequisites

- Node.js 18.x or later
- npm or yarn

### Installation

1. Install dependencies:

```bash
cd ui
npm install
```

2. Start the development server:

```bash
npm run dev
```

3. Open [http://localhost:3000](http://localhost:3000) in your browser

## API Integration

The UI communicates with the Stathera API server, which provides endpoints for:

- Account management
- Transaction processing
- System information
- Time oracle data

The API client is implemented in `src/lib/api.ts`.

## Deployment

The UI can be deployed to Vercel with minimal configuration:

1. Push your code to a Git repository
2. Import the project in Vercel
3. Set the environment variables
4. Deploy

## Future Enhancements

- **Wallet Integration**: Connect with popular cryptocurrency wallets
- **Advanced Analytics**: More detailed system metrics and visualizations
- **Forward Contract Interface**: UI for the forward contract mechanism
- **Multi-language Support**: Internationalization for global users
- **Theme Customization**: Additional theme options and customization
