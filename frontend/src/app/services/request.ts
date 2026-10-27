import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Request {
  id: number;
  request_number: string;
  request_type: string;
  requested_by: string;
  amount: number;
  description: string;
  status: string;
  current_approval_level: number;
  rejection_reason: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalHistory {
  id: number;
  request_id: number;
  approval_level: number;
  approver_role: string;
  approver_name: string;
  action: string;
  rejection_reason: string;
  created_at: string;
}

export interface User {
  id: number;
  name: string;
  role: string;
}

export interface DashboardStats {
  total: number;
  pending: number;
  approved: number;
  rejected: number;
  cancelled: number;
  total_amount: number;
  pending_amount: number;
  approved_amount: number;
}

export interface AmountReport {
  tier: string;
  label: string;
  total: number;
  approved: number;
  rejected: number;
  pending: number;
  total_amount: number;
}

@Injectable({ providedIn: 'root' })
export class RequestService {
  private baseUrl = 'http://localhost:8081/api';

  constructor(private http: HttpClient) {}

  getDashboardStats(): Observable<DashboardStats> {
    return this.http.get<DashboardStats>(`${this.baseUrl}/dashboard`);
  }

  getAmountReport(): Observable<AmountReport[]> {
    return this.http.get<AmountReport[]>(`${this.baseUrl}/reports/amount`);
  }

  getUsers(): Observable<User[]> {
    return this.http.get<User[]>(`${this.baseUrl}/users`);
  }

  createUser(user: { name: string; role: string; password: string }): Observable<User> {
    return this.http.post<User>(`${this.baseUrl}/users`, user);
  }

  loginUser(name: string, password: string): Observable<User> {
    return this.http.post<User>(`${this.baseUrl}/users/login`, { name, password });
  }

  getRequests(filters?: { status?: string; type?: string; requested_by?: string }): Observable<Request[]> {
    let params = new HttpParams();
    if (filters?.status) params = params.set('status', filters.status);
    if (filters?.type) params = params.set('type', filters.type);
    if (filters?.requested_by) params = params.set('requested_by', filters.requested_by);
    return this.http.get<Request[]>(`${this.baseUrl}/requests`, { params });
  }

  getRequestDetail(id: number): Observable<Request> {
    return this.http.get<Request>(`${this.baseUrl}/requests/${id}`);
  }

  getApprovalHistory(id: number): Observable<ApprovalHistory[]> {
    return this.http.get<ApprovalHistory[]>(`${this.baseUrl}/requests/${id}/history`);
  }

  createRequest(request: {
    requested_by: string;
    request_type: string;
    amount: number;
    description: string;
  }): Observable<Request> {
    return this.http.post<Request>(`${this.baseUrl}/requests`, request);
  }

  submitRequest(requestId: number): Observable<any> {
    return this.http.post(`${this.baseUrl}/requests/submit`, { request_id: requestId });
  }

  recordDecision(decision: {
    request_id: number;
    approver_name: string;
    approver_role: string;
    action: string;
    rejection_reason: string;
  }): Observable<any> {
    return this.http.post(`${this.baseUrl}/requests/decision`, decision);
  }
}
