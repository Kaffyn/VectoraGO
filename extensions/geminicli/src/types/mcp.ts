// MCP (Model Context Protocol) TypeScript type definitions.
// Based on https://modelcontextprotocol.io

// ---- JSON-RPC 2.0 Base ----

export interface JsonRpcRequest {
  jsonrpc: '2.0';
  id: number;
  method: string;
  params?: any;
}

export interface JsonRpcResponse {
  jsonrpc: '2.0';
  id: number;
  result?: any;
  error?: JsonRpcError;
}

export interface JsonRpcError {
  code: number;
  message: string;
  data?: any;
}

// ---- Initialization ----

export interface InitializeRequest {
  protocolVersion: string;
  capabilities: ClientCapabilities;
  clientInfo: {
    name: string;
    version: string;
  };
}

export interface ClientCapabilities {
  roots?: { listChanged?: boolean };
  sampling?: {};
}

export interface InitializeResponse {
  protocolVersion: string;
  capabilities: ServerCapabilities;
  serverInfo: {
    name: string;
    version: string;
  };
}

export interface ServerCapabilities {
  tools?: {};
  resources?: { subscribe?: boolean; listChanged?: boolean };
  prompts?: { listChanged?: boolean };
}

// ---- Tools ----

export interface ToolsListRequest {
  cursor?: string;
}

export interface ToolsListResponse {
  tools: Tool[];
  nextCursor?: string;
}

export interface Tool {
  name: string;
  description: string;
  inputSchema: {
    type: 'object';
    properties: Record<string, ToolProperty>;
    required?: string[];
  };
}

export interface ToolProperty {
  type: string;
  description?: string;
}

export interface ToolCallRequest {
  name: string;
  arguments: Record<string, any>;
}

export interface ToolCallResponse {
  content: ToolContent[];
  isError?: boolean;
}

export interface ToolContent {
  type: 'text' | 'image' | 'resource';
  text?: string;
  data?: string;
  mimeType?: string;
  resource?: {
    uri: string;
    mimeType: string;
    text?: string;
    blob?: string;
  };
}

// ---- Resources ----

export interface ResourcesListRequest {
  cursor?: string;
}

export interface ResourcesListResponse {
  resources: Resource[];
  nextCursor?: string;
}

export interface Resource {
  uri: string;
  name: string;
  description?: string;
  mimeType?: string;
}

export interface ResourceReadRequest {
  uri: string;
}

export interface ResourceReadResponse {
  contents: ResourceContent[];
}

export interface ResourceContent {
  uri: string;
  mimeType: string;
  text?: string;
  blob?: string;
}

// ---- Prompts ----

export interface PromptsListRequest {
  cursor?: string;
}

export interface PromptsListResponse {
  prompts: Prompt[];
  nextCursor?: string;
}

export interface Prompt {
  name: string;
  description?: string;
  arguments?: PromptArgument[];
}

export interface PromptArgument {
  name: string;
  description?: string;
  required?: boolean;
}

export interface PromptGetRequest {
  name: string;
  arguments?: Record<string, string>;
}

export interface PromptGetResponse {
  description?: string;
  messages: PromptMessage[];
}

export interface PromptMessage {
  role: 'user' | 'assistant';
  content: {
    type: string;
    text?: string;
  };
}
