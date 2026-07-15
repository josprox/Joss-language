use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::io::{self, Read};

pub const PROTOCOL: &str = "joss-rpc-v1";
pub type Method = Box<dyn Fn(Vec<Value>) -> Result<Value, String>>;

#[derive(Deserialize)]
struct Request {
    protocol: String,
    id: String,
    method: String,
    #[serde(default)]
    args: Vec<Value>,
}

#[derive(Serialize)]
struct RpcError {
    code: String,
    message: String,
}

#[derive(Serialize)]
struct Response {
    id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<RpcError>,
}

pub fn run(methods: HashMap<String, Method>) -> Result<(), Box<dyn std::error::Error>> {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input)?;
    let request: Request = serde_json::from_str(&input)?;
    let outcome = if request.protocol != PROTOCOL {
        Err("unsupported protocol".to_string())
    } else if let Some(method) = methods.get(&request.method) {
        method(request.args)
    } else {
        Err(format!("unknown method: {}", request.method))
    };
    let response = match outcome {
        Ok(result) => Response { id: request.id, result: Some(result), error: None },
        Err(message) => Response {
            id: request.id,
            result: None,
            error: Some(RpcError { code: "PLUGIN_ERROR".into(), message }),
        },
    };
    println!("{}", serde_json::to_string(&response)?);
    Ok(())
}
