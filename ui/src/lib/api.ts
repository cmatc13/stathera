import axios from 'axios';

// Create an axios instance with default config
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface Account {
  address: string;
  balance: number;
  publicKey: string;
}

export interface Transaction {
  id: string;
  sender: string;
  receiver: string;
  amount: number;
  fee: number;
  type: string;
  status: string;
  timestamp: string;
  description?: string;
}

export interface SystemInfo {
  totalSupply: number;
  minInflation: number;
  maxInflation: number;
  currentInflation: number;
}

export interface TimeInfo {
  timestamp: number;
  proof: string;
}

// API functions
export const getAccounts = async (): Promise<Account[]> => {
  const response = await api.get('/accounts');
  return response.data;
};

export const getAccount = async (address: string): Promise<Account> => {
  const response = await api.get(`/accounts/${address}`);
  return response.data;
};

export const createAccount = async (address: string): Promise<Account> => {
  const response = await api.post('/accounts', { address });
  return response.data;
};

export const getBalance = async (address: string): Promise<number> => {
  const response = await api.get(`/accounts/${address}/balance`);
  return response.data.balance;
};

export const getTransactions = async (): Promise<Transaction[]> => {
  const response = await api.get('/transactions');
  return response.data;
};

export const getTransaction = async (id: string): Promise<Transaction> => {
  const response = await api.get(`/transactions/${id}`);
  return response.data;
};

export const submitTransaction = async (
  sender: string,
  receiver: string,
  amount: number,
  fee: number,
  type: string,
  nonce: string,
  description?: string,
  signature?: string
): Promise<{ id: string; status: string }> => {
  const response = await api.post('/transactions', {
    sender,
    receiver,
    amount,
    fee,
    type,
    nonce,
    description,
    signature,
  });
  return response.data;
};

export const getSystemInfo = async (): Promise<SystemInfo> => {
  const supplyResponse = await api.get('/system/supply');
  const inflationResponse = await api.get('/system/inflation');
  
  return {
    totalSupply: supplyResponse.data.total_supply,
    minInflation: inflationResponse.data.min_inflation,
    maxInflation: inflationResponse.data.max_inflation,
    currentInflation: inflationResponse.data.current_inflation,
  };
};

export const getTimeInfo = async (): Promise<TimeInfo> => {
  const response = await api.get('/system/time');
  return {
    timestamp: response.data.timestamp,
    proof: response.data.proof,
  };
};

export default api;
