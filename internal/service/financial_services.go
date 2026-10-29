package service

import (
	"context"

	"chit-fund-api/internal/domain"
	"chit-fund-api/internal/repository"
)

type FinancialService struct {
	repo *repository.FinancialRepository
}

func NewFinancialService(repo *repository.FinancialRepository) *FinancialService {
	return &FinancialService{repo: repo}
}

func (s *FinancialService) GetProfitLoss(ctx context.Context, companyID int, req domain.FinancialStatementRequest) (*domain.ProfitLossStatement, error) {
	return s.repo.GetProfitLoss(ctx, companyID, req)
}

func (s *FinancialService) GetBalanceSheet(ctx context.Context, companyID int, req domain.FinancialStatementRequest) (*domain.BalanceSheet, error) {
	return s.repo.GetBalanceSheet(ctx, companyID, req)
}

func (s *FinancialService) GetCashFlow(ctx context.Context, companyID int, req domain.FinancialStatementRequest) (*domain.CashFlowStatement, error) {
	return s.repo.GetCashFlow(ctx, companyID, req)
}

func (s *FinancialService) GetCreditorDebtorSchedule(ctx context.Context, companyID int, req domain.FinancialStatementRequest) (*domain.CreditorDebtorSchedule, error) {
	return s.repo.GetCreditorDebtorSchedule(ctx, companyID, req)
}

func (s *FinancialService) GetGSTSummary(ctx context.Context, companyID int, req domain.FinancialStatementRequest) (*domain.GSTSummary, error) {
	return s.repo.GetGSTSummary(ctx, companyID, req)
}

func (s *FinancialService) CreateExpense(ctx context.Context, companyID, userID int, req *domain.CreateExpenseRequest) (*domain.CompanyExpense, error) {
	return s.repo.CreateExpense(ctx, companyID, userID, req)
}

func (s *FinancialService) ListExpenses(ctx context.Context, companyID int, req domain.FinancialStatementRequest, category string, limit, offset int) ([]*domain.CompanyExpense, int64, error) {
	from, to, _, err := s.repo.ResolvePeriodPublic(req)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListExpenses(ctx, companyID, from, to, category, limit, offset)
}

func (s *FinancialService) CreateAsset(ctx context.Context, companyID, userID int, req *domain.CreateAssetRequest) (*domain.CompanyAsset, error) {
	return s.repo.CreateAsset(ctx, companyID, userID, req)
}

func (s *FinancialService) CreateCapital(ctx context.Context, companyID, userID int, req *domain.CreateCapitalRequest) (*domain.CompanyCapital, error) {
	return s.repo.CreateCapital(ctx, companyID, userID, req)
}
