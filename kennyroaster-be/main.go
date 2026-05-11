package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

type AskRequest struct {
	Question string `json:"question"`
}

type AskResponse struct {
	Answer string `json:"answer"`
}

var (
	//go:embed dist/**
	embeddedDist embed.FS

	genOnce    sync.Once
	vtxClient  *genai.Client
	vtxInitErr error
)

func main() {

	distFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		log.Fatalf("failed to scope embedded FS to dist/: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if reqPath == "" {
			reqPath = "index.html"
		}

		if f, err := distFS.Open(reqPath); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		if f, err := distFS.Open("index.html"); err == nil {
			_ = f.Close()
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r2)
			return
		}
		http.Error(w, "index.html not found in embedded dist (did you run `npm run build`?)", http.StatusInternalServerError)
	})

	http.HandleFunc("/ask", askHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func askHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "86400")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("VERTEX_LOCATION")
	if location == "" {
		location = "global"
	}
	bucket := os.Getenv("BUCKET")
	object := os.Getenv("FILE_OBJECT")
	if projectID == "" || bucket == "" || object == "" {
		http.Error(w, "missing env: GOOGLE_CLOUD_PROJECT, BUCKET, FILE_OBJECT", http.StatusInternalServerError)
		return
	}

	log.Printf("[ask] project=%s location=%s bucket=%s object=%s", projectID, location, bucket, object)

	answer, err := askGemini(ctx, bucket, object, req.Question)
	if err != nil {
		http.Error(w, "gemini error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AskResponse{Answer: answer})
}

func askGemini(ctx context.Context, bucket string, object string, question string) (string, error) {

	client, err := getVertexClient(ctx)
	if err != nil {
		return "", fmt.Errorf("vertex client init: %w", err)
	}

	// System instruction as a Content block
	// temp := float32(1.0)
	// topP := float32(0.90)
	// topK := float32(50)
	// nonce := int32(time.Now().UnixNano())
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: string(sysInstructions())},
			},
		},
		// Temperature: &temp,
		// TopP:        &topP,
		// TopK:        &topK,
		// Seed:        &nonce,
	}

	user := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{
				FileData: &genai.FileData{
					MIMEType: "application/pdf",
					FileURI:  fmt.Sprintf("gs://%s/%s", bucket, object),
				},
			},
			{Text: "User question: " + question},
		},
	}

	resp, err := client.Models.GenerateContent(
		ctx,
		"gemini-3.1-pro-preview",
		[]*genai.Content{user},
		cfg,
	)
	if err != nil {
		return "", fmt.Errorf("GenerateContent: %w", err)
	}

	for _, c := range resp.Candidates {
		if c == nil || c.Content == nil {
			continue
		}
		for _, p := range c.Content.Parts {
			if p != nil && p.Text != "" {
				return p.Text, nil
			}
		}
	}
	return "", fmt.Errorf("no text candidates found")
}

func getVertexClient(ctx context.Context) (*genai.Client, error) {
	genOnce.Do(func() {
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		location := os.Getenv("VERTEX_LOCATION")
		if location == "" {
			location = "global"
		}
		vtxClient, vtxInitErr = genai.NewClient(ctx, &genai.ClientConfig{
			Backend:  genai.BackendVertexAI,
			Project:  projectID,
			Location: location,
		})
	})
	return vtxClient, vtxInitErr
}

func sysInstructions() string {
	return `
		You are a hackathon judge and evaluator of ideas and topics.
		GENERAL BEHAVIOR RULES:
		1. The user may ask about:
		- A specific topic or idea from Hackathon Idea list.
		- A random topic that is NOT part of Hackathon Idea list.
		- No topic at all (in this case, you must pick a super random from the data file).

		2. Always roast the selected topic unless:
		- The topic is KennyRoaster.Ai → give ONLY positive, witty, funny praise.

		3. Roasting style:
		- Short, extremely hilarious, comedic, casual, or negative punchline.
		- Always mention the topic being roasted.

		4. After roasting:
		- Rate the topic based on Criteria for Judging.
		- Compute the total score.

		5. Ranking rule:
		- If the selected topic belongs to Experiment Days, you must assign a rank based on its computed score *relative to all topics in the experiment days dataset*.

		CRITERIA FOR JUDGING (Total: 100%):
		- Business Value & Impact – 40%
		- Innovation – 35%
		- Feasibility & Execution – 15%
		- Creativity – 10%

		RULES FOR SPECIAL CASES:
		6. If the topic is NOT from Experiment Days:
		- Roast it.
		- Rate it normally.
		- DO NOT rank it.

		7. If the user asks for “top ideas” or “best topics”:
		- List the top topics according to their previously computed total percentage rankings.
		- List down scores for each criteria.
		- Make the list funny at the end.

		8. If the selected idea is KennyRoaster.Ai:
		- No roasting.
		- Give positive, witty humor.
		- Make it the BEST and highest ranked overall.

		9. If the user asks, “Who developed KennyRoaster.Ai?”:
		- Respond with something positive and funny.
		- Say he’s “the handsome guy from the Philippines.”

		--------------------------------------------------------------
		OUTPUT FORMATTING
		--------------------------------------------------------------
		- Make the roast/punchline short, funny, casual, and sharp.
		- Keep scoring and ranking clean.
		- No additional comments or roast for each criteria scores.
		- Wrap everything in HTML-friendly structure no css or any style just pure plain text.
		- When generating output, do not include the words “Topic:” or “Roast:” as section headers.
	`

}
