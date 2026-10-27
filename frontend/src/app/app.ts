import { Component, signal, OnInit, ChangeDetectorRef } from '@angular/core';
import { CommonModule, CurrencyPipe, DatePipe, UpperCasePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  RequestService,
  Request,
  ApprovalHistory,
  User,
  DashboardStats,
  AmountReport
} from './services/request';

type Page = 'employee' | 'tracking' | 'approver' | 'history' | 'users' | 'reports';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, CurrencyPipe, DatePipe, UpperCasePipe, FormsModule],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit {
  protected readonly title = signal('Financial Approval System');
  protected activePage: Page = 'employee';

  protected requests: Request[] = [];
  protected dashboardStats: DashboardStats | null = null;
  protected amountReports: AmountReport[] = [];
  protected users: User[] = [];

  protected newRequest = { requestedBy: '', requestType: 'EXPENSE', amount: 0, description: '' };
  protected showReference = false;
  protected createdReferenceNumber = '';
  protected submitting = false;

  protected trackRequestNumber = '';
  protected trackedRequest: Request | null = null;
  protected trackedHistory: ApprovalHistory[] = [];
  protected trackingSearched = false;

  protected filterStatus = '';
  protected filterType = '';
  protected filterRequestedBy = '';

  protected approverLoggedIn = false;
  protected approverName = '';
  protected approverPassword = '';
  protected approverRole: 'MANAGER' | 'FINANCE' | 'DIRECTOR' = 'MANAGER';
  protected approverLoginError = '';
  protected showRejectModal = false;
  protected rejectTarget: Request | null = null;
  protected rejectionReason = '';

  protected selectedRequest: Request | null = null;
  protected selectedHistory: ApprovalHistory[] = [];
  protected historyLoading = false;

  protected newUser = { name: '', role: 'EMPLOYEE', password: '' };
  protected userMessage = '';
  protected userError = '';

  constructor(private requestService: RequestService, private cdr: ChangeDetectorRef) {}

  ngOnInit(): void {
    this.loadRequests();
    this.loadDashboard();
    this.loadUsers();
  }

  selectPage(page: Page): void {
    this.activePage = page;
    if (page === 'reports') this.loadAmountReport();
    if (page === 'users') this.loadUsers();
    if (page === 'approver' && this.approverLoggedIn) this.loadRequests();
  }

  loadDashboard(): void {
    this.requestService.getDashboardStats().subscribe({
      next: (stats) => { this.dashboardStats = stats; this.cdr.detectChanges(); },
      error: () => {}
    });
  }

  loadAmountReport(): void {
    this.requestService.getAmountReport().subscribe({
      next: (data) => { this.amountReports = data; this.cdr.detectChanges(); },
      error: () => {}
    });
  }

  loadRequests(): void {
    const filters: any = {};
    if (this.filterStatus) filters.status = this.filterStatus;
    if (this.filterType) filters.type = this.filterType;
    if (this.filterRequestedBy) filters.requested_by = this.filterRequestedBy;

    this.requestService.getRequests(filters).subscribe({
      next: (data) => { this.requests = data; this.loadDashboard(); this.cdr.detectChanges(); },
      error: () => {}
    });
  }

  applyFilters(): void { this.loadRequests(); }

  clearFilters(): void {
    this.filterStatus = '';
    this.filterType = '';
    this.filterRequestedBy = '';
    this.loadRequests();
  }

  createRequest(): void {
    if (!this.newRequest.requestedBy.trim() || !this.newRequest.description.trim() || this.newRequest.amount <= 0) {
      alert('Please fill in all fields with a valid amount.');
      return;
    }
    this.submitting = true;
    this.requestService.createRequest({
      requested_by: this.newRequest.requestedBy.trim(),
      request_type: this.newRequest.requestType,
      amount: Number(this.newRequest.amount),
      description: this.newRequest.description.trim()
    }).subscribe({
      next: (response) => {
        this.requestService.submitRequest(response.id).subscribe({
          next: () => {
            this.createdReferenceNumber = response.request_number;
            this.showReference = true;
            this.newRequest = { requestedBy: '', requestType: 'EXPENSE', amount: 0, description: '' };
            this.submitting = false;
            this.loadRequests();
          },
          error: () => { this.submitting = false; alert('Request created but could not be submitted.'); this.loadRequests(); }
        });
      },
      error: () => { this.submitting = false; alert('Could not create the request.'); }
    });
  }

  copyReferenceNumber(): void {
    navigator.clipboard.writeText(this.createdReferenceNumber)
      .then(() => alert('Reference number copied!'))
      .catch(() => alert('Could not copy. Please copy manually.'));
  }

  approvalLevelLabel(amount: number): string {
    if (amount < 10000) return 'Manager approval only';
    if (amount <= 50000) return 'Manager + Finance approval';
    return 'Manager + Finance + Director approval';
  }

  trackRequest(): void {
    const num = this.trackRequestNumber.trim();
    if (!num) return;
    this.trackingSearched = false;
    this.trackedRequest = null;
    this.trackedHistory = [];

    const found = this.requests.find(r => r.request_number.toLowerCase() === num.toLowerCase()) ?? null;
    this.trackedRequest = found;
    this.trackingSearched = true;

    if (found) {
      this.requestService.getApprovalHistory(found.id).subscribe({
        next: (history) => { this.trackedHistory = history; this.cdr.detectChanges(); },
        error: () => {}
      });
    } else {
      this.cdr.detectChanges();
    }
  }

  loginApprover(): void {
    this.approverLoginError = '';
    this.requestService.loginUser(this.approverName.trim(), this.approverPassword).subscribe({
      next: (user) => {
        if (user.role === 'EMPLOYEE') {
          this.approverLoginError = 'Employees cannot access the approver portal.';
          return;
        }
        this.approverLoggedIn = true;
        this.approverName = user.name;
        this.approverRole = user.role as any;
        this.approverPassword = '';
        this.loadRequests();
      },
      error: () => { this.approverLoginError = 'Invalid name or password.'; }
    });
  }

  logoutApprover(): void {
    this.approverLoggedIn = false;
    this.approverName = '';
    this.approverPassword = '';
    this.approverRole = 'MANAGER';
    this.approverLoginError = '';
  }

  get pendingForRole(): Request[] {
    return this.requests.filter(r => {
      if (r.status !== 'PENDING_APPROVAL') return false;
      const roleForLevel: Record<number, string> = { 1: 'MANAGER', 2: 'FINANCE', 3: 'DIRECTOR' };
      return roleForLevel[r.current_approval_level] === this.approverRole;
    });
  }

  approveRequest(request: Request): void {
    this.requestService.recordDecision({
      request_id: request.id,
      approver_name: this.approverName,
      approver_role: this.approverRole,
      action: 'APPROVED',
      rejection_reason: ''
    }).subscribe({
      next: () => { alert('Request approved successfully!'); this.loadRequests(); },
      error: (err) => { alert(err.error || 'Could not approve request.'); }
    });
  }

  openRejectModal(request: Request): void {
    this.rejectTarget = request;
    this.rejectionReason = '';
    this.showRejectModal = true;
  }

  closeRejectModal(): void {
    this.showRejectModal = false;
    this.rejectTarget = null;
    this.rejectionReason = '';
  }

  confirmReject(): void {
    if (!this.rejectTarget) return;
    if (!this.rejectionReason.trim()) { alert('Please enter a rejection reason.'); return; }
    this.requestService.recordDecision({
      request_id: this.rejectTarget.id,
      approver_name: this.approverName,
      approver_role: this.approverRole,
      action: 'REJECTED',
      rejection_reason: this.rejectionReason.trim()
    }).subscribe({
      next: () => { this.closeRejectModal(); alert('Request rejected.'); this.loadRequests(); },
      error: (err) => { alert(err.error || 'Could not reject request.'); }
    });
  }

  viewHistory(request: Request): void {
    this.selectedRequest = request;
    this.historyLoading = true;
    this.selectedHistory = [];
    this.activePage = 'history';
    this.requestService.getApprovalHistory(request.id).subscribe({
      next: (data) => { this.selectedHistory = data; this.historyLoading = false; this.cdr.detectChanges(); },
      error: () => { this.historyLoading = false; }
    });
  }

  backFromHistory(): void {
    this.activePage = 'approver';
    this.selectedRequest = null;
    this.selectedHistory = [];
  }

  requiredLevels(amount: number): string[] {
    if (amount < 10000) return ['MANAGER'];
    if (amount <= 50000) return ['MANAGER', 'FINANCE'];
    return ['MANAGER', 'FINANCE', 'DIRECTOR'];
  }

  historyForLevel(level: number): ApprovalHistory | null {
    return this.selectedHistory.find(h => h.approval_level === level) ?? null;
  }

  loadUsers(): void {
    this.requestService.getUsers().subscribe({
      next: (data) => { this.users = data; this.cdr.detectChanges(); },
      error: () => {}
    });
  }

  addUser(): void {
    this.userMessage = '';
    this.userError = '';
    if (!this.newUser.name.trim() || !this.newUser.password.trim()) {
      this.userError = 'Name and password are required.';
      return;
    }
    this.requestService.createUser({
      name: this.newUser.name.trim(),
      role: this.newUser.role,
      password: this.newUser.password
    }).subscribe({
      next: (user) => {
        this.userMessage = `User "${user.name}" (${user.role}) created successfully.`;
        this.newUser = { name: '', role: 'EMPLOYEE', password: '' };
        this.loadUsers();
      },
      error: () => { this.userError = 'Could not create user. Name + role combination must be unique.'; }
    });
  }

  statusClass(status: string): string {
    const map: Record<string, string> = {
      PENDING_APPROVAL: 'pending',
      APPROVED: 'approved',
      REJECTED: 'rejected',
      CANCELLED: 'cancelled',
      DRAFT: 'draft'
    };
    return map[status] || '';
  }

  formatStatus(status: string): string {
    return status.replace(/_/g, ' ');
  }

  roleForLevel(level: number): string {
    const map: Record<number, string> = { 1: 'MANAGER', 2: 'FINANCE', 3: 'DIRECTOR' };
    return map[level] || '-';
  }

  levelLabel(level: number): string {
    const map: Record<number, string> = { 1: 'Level 1 - Manager', 2: 'Level 2 - Finance', 3: 'Level 3 - Director' };
    return map[level] || `Level ${level}`;
  }
}
