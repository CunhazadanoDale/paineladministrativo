package postgres

import (
	"context"
	"strconv"

	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type linhaResumoLinha struct {
	EmpresaID         uuid.UUID `db:"empresa_id"`
	RepresentanteID   uuid.UUID `db:"representante_id"`
	RepresentanteNome string    `db:"representante_nome"`
	ComissoesCentavos int64     `db:"comissoes_centavos"`
	AjustesCentavos   int64     `db:"ajustes_centavos"`
	EmAbertoCentavos  int64     `db:"em_aberto_centavos"`
	FechadoCentavos   int64     `db:"fechado_centavos"`
	PagoCentavos      int64     `db:"pago_centavos"`
}

func (r *ComissaoRepository) Resumo(ctx context.Context, filtro portsout.ResumoFiltro) ([]domaincomissao.LinhaResumo, error) {
	condicoes := []string{"competencia = $1"}
	argumentos := []any{filtro.Competencia.Inicio()}
	adicionar := func(expressao string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, expressao+" $"+strconv.Itoa(len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("empresa_id =", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.RepresentanteID != nil {
		adicionar("representante_id =", *filtro.RepresentanteID)
	}
	onde := unirCondicoes(condicoes)

	var linhas []*linhaResumoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT i.empresa_id, i.representante_id, rep.nome AS representante_nome,
			COALESCE(SUM(i.valor) FILTER (WHERE i.tipo = 'comissao'), 0)::bigint AS comissoes_centavos,
			COALESCE(SUM(i.valor) FILTER (WHERE i.tipo = 'ajuste'), 0)::bigint AS ajustes_centavos,
			COALESCE(SUM(i.valor) FILTER (WHERE i.fechamento_id IS NULL), 0)::bigint AS em_aberto_centavos,
			COALESCE(SUM(i.valor) FILTER (WHERE i.fechamento_id IS NOT NULL), 0)::bigint AS fechado_centavos,
			COALESCE(SUM(i.valor) FILTER (WHERE s.status = 'pago'), 0)::bigint AS pago_centavos
		FROM (
			SELECT 'comissao' AS tipo, empresa_id, representante_id, valor_centavos AS valor, fechamento_id
			FROM comissao
			WHERE `+onde+`
			UNION ALL
			SELECT 'ajuste', empresa_id, representante_id, valor_centavos, fechamento_id
			FROM comissao_ajuste
			WHERE `+onde+`
		) i
		JOIN representante rep ON rep.id = i.representante_id
		LEFT JOIN comissao_fechamento f ON f.id = i.fechamento_id
		LEFT JOIN solicitacao s ON s.id = f.solicitacao_id
		GROUP BY i.empresa_id, i.representante_id, rep.nome
		ORDER BY rep.nome, i.representante_id, i.empresa_id
	`, argumentos...); err != nil {
		return nil, tratarErroDeConsulta(err)
	}

	resumo := make([]domaincomissao.LinhaResumo, 0, len(linhas))
	for _, linha := range linhas {
		resumo = append(resumo, domaincomissao.LinhaResumo{
			EmpresaID:         linha.EmpresaID,
			RepresentanteID:   linha.RepresentanteID,
			RepresentanteNome: linha.RepresentanteNome,
			ComissoesCentavos: linha.ComissoesCentavos,
			AjustesCentavos:   linha.AjustesCentavos,
			EmAbertoCentavos:  linha.EmAbertoCentavos,
			FechadoCentavos:   linha.FechadoCentavos,
			PagoCentavos:      linha.PagoCentavos,
		})
	}

	return resumo, nil
}
