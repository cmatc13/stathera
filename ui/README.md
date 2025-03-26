# Stathera UI

This is the frontend user interface for the Stathera monetary system. It provides a modern, responsive web interface for interacting with the Stathera API.

## Features

- Dashboard with system metrics and account overview
- Account management (create, view, export)
- Transaction submission and history
- System information and metrics
- Responsive design for desktop and mobile

## Technology Stack

- **Next.js**: React framework for server-side rendering and static site generation
- **TypeScript**: Type-safe JavaScript
- **Tailwind CSS**: Utility-first CSS framework
- **Radix UI**: Accessible UI components
- **SWR**: React Hooks for data fetching
- **Axios**: Promise-based HTTP client
- **Recharts**: Composable charting library

## Getting Started

### Prerequisites

- Node.js 18.x or later
- npm or yarn

### Installation

1. Install dependencies:

```bash
cd ui
npm install
# or
yarn install
```

2. Set up environment variables:

Create a `.env.local` file in the root directory with the following variables:

```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

### Development

Run the development server:

```bash
npm run dev
# or
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser to see the application.

### Building for Production

Build the application for production:

```bash
npm run build
# or
yarn build
```

Start the production server:

```bash
npm run start
# or
yarn start
```

## Project Structure

```
ui/
├── public/              # Static assets
├── src/
│   ├── app/             # Next.js app router pages
│   ├── components/      # Reusable UI components
│   ├── lib/             # Utility functions and API client
│   └── types/           # TypeScript type definitions
├── .env.local           # Environment variables (create this file)
├── next.config.js       # Next.js configuration
├── package.json         # Project dependencies and scripts
├── postcss.config.js    # PostCSS configuration
├── tailwind.config.ts   # Tailwind CSS configuration
└── tsconfig.json        # TypeScript configuration
```

## API Integration

The UI communicates with the Stathera API server, which provides endpoints for:

- Account management
- Transaction processing
- System information
- Time oracle data

See the `src/lib/api.ts` file for the API client implementation.

## Deployment

The UI can be deployed to Vercel with minimal configuration:

1. Push your code to a Git repository
2. Import the project in Vercel
3. Set the environment variables
4. Deploy

## License

This project is licensed under the MIT License - see the LICENSE file for details.
