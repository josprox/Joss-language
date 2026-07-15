package files

import (
	"path/filepath"
)

func GetBrunoFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "bruno", "bruno.json"): `{
  "version": "1",
  "name": "Joss API",
  "type": "collection"
}`,
		filepath.Join(path, "bruno", "1. Register.bru"): `meta {
  name: Register
  type: http
  seq: 1
}

post {
  url: http://localhost:8000/api/register
  body: json
  auth: none
}

body:json {
  {
    "first_name": "Bruno",
    "last_name": "Tester",
    "username": "brunotest",
    "email": "bruno@example.com",
    "password": "password123",
    "phone": "5551234567"
  }
}`,
		filepath.Join(path, "bruno", "2. Login.bru"): `meta {
  name: Login
  type: http
  seq: 2
}

post {
  url: http://localhost:8000/api/login
  body: json
  auth: none
}

body:json {
  {
    "email": "bruno@example.com",
    "password": "password123"
  }
}

script:post-response {
  if (res.status === 200) {
    bru.setVar("token", res.body.token);
  }
}`,
		filepath.Join(path, "bruno", "3. Refresh Token.bru"): `meta {
  name: Refresh Token
  type: http
  seq: 3
}

post {
  url: http://localhost:8000/api/refresh
  body: none
  auth: bearer
}

auth:bearer {
  token: {{token}}
}

script:post-response {
  if (res.status === 200) {
    bru.setVar("token", res.body.token);
  }
}`,
		filepath.Join(path, "bruno", "4. Forgot Password.bru"): `meta {
  name: Forgot Password
  type: http
  seq: 4
}

post {
  url: http://localhost:8000/password/email
  body: json
  auth: none
}

body:json {
  {
    "email": "bruno@example.com"
  }
}`,
		filepath.Join(path, "bruno", "5. Reset Password.bru"): `meta {
  name: Reset Password
  type: http
  seq: 5
}

post {
  url: http://localhost:8000/password/reset
  body: json
  auth: none
}

body:json {
  {
    "token": "TOKEN_FROM_EMAIL",
    "password": "newpassword123"
  }
}`,
		filepath.Join(path, "bruno", "6. Delete Account.bru"): `meta {
  name: Delete Account
  type: http
  seq: 6
}

delete {
  url: http://localhost:8000/api/delete
  body: none
  auth: bearer
}

auth:bearer {
  token: {{token}}
}`,
	}
}
