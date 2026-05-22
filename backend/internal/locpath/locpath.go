// Package locpath constrói o caminho completo (full path) de uma localização
// percorrendo a hierarquia parent_id. Usado tanto pelo HTTP handler quanto pelo
// MCP server para enriquecer respostas com strings legíveis tipo
// "Garagem > Caixa Ferramentas".
package locpath

import "database/sql"

type locNode struct {
	name   string
	parent *int64
}

// LoadLocationMap carrega todas as locations em memória num map id->{name,parent_id}.
// Usado para resolver paths em lote sem fazer N queries.
func LoadLocationMap(db *sql.DB) map[int64]locNode {
	out := map[int64]locNode{}
	rows, err := db.Query("SELECT id, name, parent_id FROM locations")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var n locNode
		if err := rows.Scan(&id, &n.name, &n.parent); err == nil {
			out[id] = n
		}
	}
	return out
}

// BuildFullPathFromMap monta o caminho usando um mapa pré-carregado.
// Resultado: "Garagem > Caixa Ferramentas > Gaveta Esquerda".
func BuildFullPathFromMap(m map[int64]locNode, id int64) string {
	path := ""
	cur := &id
	for cur != nil {
		node, ok := m[*cur]
		if !ok {
			break
		}
		if path == "" {
			path = node.name
		} else {
			path = node.name + " > " + path
		}
		cur = node.parent
	}
	return path
}

// BuildFullPath é o atalho que carrega o mapa e resolve um único id.
// Para resolver múltiplos ids, prefira LoadLocationMap + BuildFullPathFromMap.
func BuildFullPath(db *sql.DB, id int64) string {
	return BuildFullPathFromMap(LoadLocationMap(db), id)
}
