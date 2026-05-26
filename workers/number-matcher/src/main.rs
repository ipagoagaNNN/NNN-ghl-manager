use axum::{extract::Json, routing::post, Router};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};

#[derive(Debug, Deserialize)]
struct MatchRequest {
    dialpad: Vec<DialpadNumber>,
    number_verifier: Vec<NVNumber>,
    hiya: Vec<HiyaNumber>,
}

#[derive(Debug, Deserialize)]
struct DialpadNumber {
    number: String,
    office_name: Option<String>,
    department_name: Option<String>,
    assigned_type: Option<String>,
    assignment_name: Option<String>,
    number_status: Option<String>,
    reserved_reason: Option<String>,
    is_reserved_number: Option<bool>,
}

#[derive(Debug, Deserialize)]
struct NVNumber {
    number: String,
}

#[derive(Debug, Deserialize)]
struct HiyaNumber {
    number: String,
    spam_label: Option<String>,
    hiya_number: Option<String>,
}

#[derive(Debug, Serialize)]
struct MatchedItem {
    number: String,
    hiya_number: Option<String>,
    hiya_spam_label: Option<String>,
    department_name: Option<String>,
    office_name: Option<String>,
    assigned_type: Option<String>,
    assignment_name: Option<String>,
    number_status: Option<String>,
    reserved_reason: Option<String>,
    is_reserved_number: bool,
    in_number_verifier: bool,
    in_hiya: bool,
}

#[derive(Debug, Serialize)]
struct MatchResponse {
    items: Vec<MatchedItem>,
    total_count: usize,
    matched_count: usize,
    unmatched_count: usize,
    office_count: usize,
}

fn compact_phone(number: &str) -> String {
    number.chars().filter(|c| c.is_ascii_digit()).collect()
}

async fn match_numbers(Json(req): Json<MatchRequest>) -> Json<MatchResponse> {
    let nv_set: HashSet<String> = req.number_verifier.iter()
        .map(|n| compact_phone(&n.number))
        .collect();

    let hiya_map: HashMap<String, &HiyaNumber> = req.hiya.iter()
        .map(|n| (compact_phone(&n.number), n))
        .collect();

    let mut office_set = HashSet::new();
    let mut matched_count = 0;

    let items: Vec<MatchedItem> = req.dialpad.iter().map(|d| {
        let key = compact_phone(&d.number);
        let in_nv = nv_set.contains(&key);
        let hiya_entry = hiya_map.get(&key);

        if in_nv { matched_count += 1; }
        if let Some(office) = &d.office_name { office_set.insert(office.clone()); }

        MatchedItem {
            number: d.number.clone(),
            hiya_number: hiya_entry.and_then(|h| h.hiya_number.clone()),
            hiya_spam_label: hiya_entry.and_then(|h| h.spam_label.clone()),
            department_name: d.department_name.clone(),
            office_name: d.office_name.clone(),
            assigned_type: d.assigned_type.clone(),
            assignment_name: d.assignment_name.clone(),
            number_status: d.number_status.clone(),
            reserved_reason: d.reserved_reason.clone(),
            is_reserved_number: d.is_reserved_number.unwrap_or(false),
            in_number_verifier: in_nv,
            in_hiya: hiya_entry.is_some(),
        }
    }).collect();

    let total = items.len();
    Json(MatchResponse {
        matched_count,
        unmatched_count: total.saturating_sub(matched_count),
        office_count: office_set.len(),
        total_count: total,
        items,
    })
}

#[tokio::main]
async fn main() {
    let app = Router::new().route("/match", post(match_numbers));
    let listener = tokio::net::TcpListener::bind("127.0.0.1:8081").await.unwrap();
    println!("number-matcher listening on 127.0.0.1:8081");
    axum::serve(listener, app).await.unwrap();
}
