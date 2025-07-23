package main

import (
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Modelo sem o campo de dificuldade
type ScoreEntry struct {
	gorm.Model
	Name  string `json:"name"`
	CPF   string `json:"cpf" gorm:"uniqueIndex"`
	Score int    `json:"score"`
}

type EasyScoreEntry struct {
	gorm.Model
	Name  string `json:"name"`
	CPF   string `json:"cpf" gorm:"uniqueIndex"`
	Score int    `json:"score"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("ranking.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Erro ao conectar ao banco:", err)
		os.Exit(1)
	}

	// Migrar modelo (cria tabela se não existir)
	if err := db.AutoMigrate(&ScoreEntry{}, &EasyScoreEntry{}); err != nil {
		log.Fatal("Erro ao migrar modelo:", err)
		os.Exit(1)
	}

	// Criar roteador Gin
	r := gin.Default()

	// Habilitar CORS (para funcionamento com o frontend)
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
	}))

	// POST /csbc - cadastrar pontuação
	r.POST("/csbc", func(c *gin.Context) {
		var input ScoreEntry
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
			return
		}

		if input.Name == "" || input.CPF == "" || input.Score < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Campos inválidos"})
			return
		}

		var existing ScoreEntry
		result := db.Where("cpf = ?", input.CPF).First(&existing)

		if result.Error == nil {
			// CPF já existe: atualizar pontuação se a nova for maior
			if input.Score > existing.Score {
				existing.Score = input.Score
				existing.Name = input.Name
				if err := db.Save(&existing).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar pontuação"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "Pontuação atualizada"})
			} else {
				c.JSON(http.StatusOK, gin.H{"message": "Pontuação menor ou igual à anterior. Nada foi alterado."})
			}
			return
		}

		if err := db.Create(&input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar no banco"})
			return
		}

		c.Status(http.StatusCreated)
	})

	// GET /csbc - retornar ranking ordenado por pontuação
	r.GET("/csbc", func(c *gin.Context) {
		var entries []ScoreEntry
		if err := db.Find(&entries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados"})
			return
		}

		// Ordenar por pontuação descrescente
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Score > entries[j].Score
		})

		c.JSON(http.StatusOK, entries)
	})

	r.DELETE("/csbc", func(c *gin.Context) {
		if err := db.Exec("DELETE FROM score_entries").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao limpar ranking"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Ranking limpo com sucesso"})
	})

	r.POST("/easyquiz", func(c *gin.Context) {
		var input EasyScoreEntry
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
			return
		}

		if input.Name == "" || input.CPF == "" || input.Score < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Campos inválidos"})
			return
		}

		var existing EasyScoreEntry
		result := db.Where("cpf = ?", input.CPF).First(&existing)

		if result.Error == nil {
			if input.Score > existing.Score {
				existing.Score = input.Score
				existing.Name = input.Name
				if err := db.Save(&existing).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar pontuação"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "Pontuação atualizada"})
			} else {
				c.JSON(http.StatusOK, gin.H{"message": "Pontuação menor ou igual à anterior. Nada foi alterado."})
			}
			return
		}

		if err := db.Create(&input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar no banco"})
			return
		}

		c.Status(http.StatusCreated)
	})

	// GET /easyquiz - retornar ranking do quiz fácil
	r.GET("/easyquiz", func(c *gin.Context) {
		var entries []EasyScoreEntry
		if err := db.Find(&entries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados"})
			return
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Score > entries[j].Score
		})

		c.JSON(http.StatusOK, entries)
	})

	// DELETE /easyquiz - limpar ranking do quiz fácil
	r.DELETE("/easyquiz", func(c *gin.Context) {
		if err := db.Exec("DELETE FROM easy_score_entries").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao limpar ranking"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Ranking fácil limpo com sucesso"})
	})

	// Iniciar servidor
	log.Println("🚀 Servidor rodando em http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}
