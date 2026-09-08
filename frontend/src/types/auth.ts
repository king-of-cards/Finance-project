export interface LoginRequest {
    user_id : string;
    password : string;
}

export interface User {
    user_id : string;
    name : string;
    role : string;
}

export interface LoginResponse {
    token : string;
    expires_at : string;
    user : User;

}