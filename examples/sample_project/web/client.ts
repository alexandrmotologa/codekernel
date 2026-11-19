export interface SessionState {
  userId: string;
  token: string;
  active: boolean;
}

export class APIClient {
  private endpoint: string;

  constructor(endpoint: string) {
    this.endpoint = endpoint;
  }

  public async fetchProfile(session: SessionState): Promise<Record<string, unknown>> {
    console.log(`Connecting to ${this.endpoint} with user ${session.userId}`);
    return { status: "active", userId: session.userId };
  }
}

export const initClient = (endpoint: string): APIClient => {
  return new APIClient(endpoint);
};
