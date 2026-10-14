package financeiro

import "github.com/google/uuid"

type RamoDoPlano struct {
	Categoria     *CategoriaFinanceira
	Subcategorias []*CategoriaFinanceira
}

func MontarPlano(categorias []*CategoriaFinanceira) []RamoDoPlano {
	posicaoDaRaiz := make(map[uuid.UUID]int)
	ramos := make([]RamoDoPlano, 0)

	for _, categoria := range categorias {
		if !categoria.EhSubcategoria() {
			posicaoDaRaiz[categoria.ID] = len(ramos)
			ramos = append(ramos, RamoDoPlano{Categoria: categoria, Subcategorias: []*CategoriaFinanceira{}})
		}
	}

	for _, categoria := range categorias {
		if !categoria.EhSubcategoria() {
			continue
		}
		if posicao, ok := posicaoDaRaiz[*categoria.CategoriaPaiID]; ok {
			ramos[posicao].Subcategorias = append(ramos[posicao].Subcategorias, categoria)
		}
	}

	return ramos
}
