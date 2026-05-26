use axum::{extract::Json, routing::post, Router};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Deserialize)]
struct ProcessRequest {
    csv_text: String,
    file_name: Option<String>,
}

#[derive(Debug, Serialize)]
struct ProcessResponse {
    headers: Vec<String>,
    rows: Vec<HashMap<String, String>>,
    row_count: usize,
    file_name: String,
    imported_at: String,
}

async fn process_csv(Json(req): Json<ProcessRequest>) -> Json<ProcessResponse> {
    let mut reader = csv::ReaderBuilder::new()
        .trim(csv::Trim::All)
        .from_reader(req.csv_text.as_bytes());

    let headers: Vec<String> = match reader.headers() {
        Ok(h) => h.iter().map(|s| s.to_string()).collect(),
        Err(_) => return Json(ProcessResponse {
            headers: vec![],
            rows: vec![],
            row_count: 0,
            file_name: req.file_name.unwrap_or_default(),
            imported_at: chrono_now(),
        }),
    };

    let mut rows = Vec::new();
    for result in reader.records() {
        if let Ok(record) = result {
            let mut row = HashMap::new();
            for (i, field) in record.iter().enumerate() {
                if let Some(header) = headers.get(i) {
                    row.insert(header.clone(), field.to_string());
                }
            }
            rows.push(row);
        }
    }

    let count = rows.len();
    Json(ProcessResponse {
        headers,
        rows,
        row_count: count,
        file_name: req.file_name.unwrap_or_else(|| "flagged_numbers.csv".to_string()),
        imported_at: chrono_now(),
    })
}

fn chrono_now() -> String {
    // Simple ISO timestamp without external chrono dep for now
    "".to_string()
}

#[tokio::main]
async fn main() {
    let app = Router::new().route("/process", post(process_csv));
    let listener = tokio::net::TcpListener::bind("127.0.0.1:8082").await.unwrap();
    println!("csv-processor listening on 127.0.0.1:8082");
    axum::serve(listener, app).await.unwrap();
}
