package lead

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.ImportacaoUseCase = (*LeadImportacaoUsecaseImpl)(nil)

type LeadImportacaoUsecaseImpl struct {
	leads  portsout.LeadImportacaoRepository
	etapas portsout.EtapaRepository
}

func NewLeadImportacaoUsecase(leads portsout.LeadImportacaoRepository, etapas portsout.EtapaRepository) *LeadImportacaoUsecaseImpl {
	return &LeadImportacaoUsecaseImpl{leads: leads, etapas: etapas}
}

type importacaoDeLeads struct {
	etapaID              uuid.UUID
	simular              bool
	emailsCadastrados    map[string]bool
	telefonesCadastrados map[string]bool
	emailsDoArquivo      map[string]int
	telefonesDoArquivo   map[string]int
}

func (u *LeadImportacaoUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarLeadsInput) (*domain.RelatorioImportacao, error) {
	relatorio, err := domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
	if err != nil {
		return nil, err
	}
	if input.EtapaID == uuid.Nil {
		return nil, domain.ErroValidacao("etapa dos leads é obrigatória")
	}
	if _, err := etapaAtiva(ctx, u.etapas, input.EtapaID, "etapa dos leads"); err != nil {
		return nil, err
	}

	emails, telefones := contatosDasLinhas(input.Linhas)
	estado := &importacaoDeLeads{
		etapaID:            input.EtapaID,
		simular:            input.Simular,
		emailsDoArquivo:    map[string]int{},
		telefonesDoArquivo: map[string]int{},
	}
	if estado.emailsCadastrados, err = u.leads.EmailsCadastrados(ctx, emails); err != nil {
		return nil, err
	}
	if estado.telefonesCadastrados, err = u.leads.TelefonesCadastrados(ctx, telefones); err != nil {
		return nil, err
	}

	for _, linha := range input.Linhas {
		motivo, err := u.importarLinha(ctx, estado, linha)
		if err != nil {
			return nil, err
		}
		if motivo != "" {
			relatorio.Rejeitar(linha.Numero, strings.TrimSpace(linha.Nome), motivo)
			continue
		}
		relatorio.Importadas++
	}

	return relatorio, nil
}

func (u *LeadImportacaoUsecaseImpl) importarLinha(ctx context.Context, estado *importacaoDeLeads, linha portsin.LinhaImportacao) (string, error) {
	lead := &domainlead.Lead{
		Nome:     linha.Nome,
		Email:    linha.Email,
		Telefone: linha.Telefone,
		Origem:   linha.Origem,
		EtapaID:  estado.etapaID,
	}
	normalizarLead(lead)
	if err := validarCadastro(lead); err != nil {
		return domain.MotivoDeRejeicao(err)
	}

	telefone := digitosDoTelefone(lead.Telefone)
	if lead.Email != "" {
		if anterior, repetido := estado.emailsDoArquivo[lead.Email]; repetido {
			return "email repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
		}
		if estado.emailsCadastrados[lead.Email] {
			return "já existe um lead com este email", nil
		}
	}
	if telefone != "" {
		if anterior, repetido := estado.telefonesDoArquivo[telefone]; repetido {
			return "telefone repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
		}
		if estado.telefonesCadastrados[telefone] {
			return "já existe um lead com este telefone", nil
		}
	}

	if !estado.simular {
		agora := time.Now().UTC()
		lead.ID = uuid.New()
		lead.Ativo = true
		lead.CriadoEm = agora
		lead.AtualizadoEm = agora
		if _, err := u.leads.Criar(ctx, lead); err != nil {
			return domain.MotivoDeRejeicao(err)
		}
	}

	if lead.Email != "" {
		estado.emailsDoArquivo[lead.Email] = linha.Numero
	}
	if telefone != "" {
		estado.telefonesDoArquivo[telefone] = linha.Numero
	}

	return "", nil
}

func contatosDasLinhas(linhas []portsin.LinhaImportacao) ([]string, []string) {
	emails := make([]string, 0, len(linhas))
	telefones := make([]string, 0, len(linhas))
	for _, linha := range linhas {
		if email := strings.ToLower(strings.TrimSpace(linha.Email)); email != "" {
			emails = append(emails, email)
		}
		if telefone := digitosDoTelefone(linha.Telefone); telefone != "" {
			telefones = append(telefones, telefone)
		}
	}

	return emails, telefones
}

func digitosDoTelefone(telefone string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, telefone)
}
