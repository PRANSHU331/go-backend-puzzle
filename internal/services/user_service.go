package services

type UserService struct{}

func NewUserService() *UserService {
    return &UserService{}
}

func GetUsers() []string {
    return []string{"Alice", "Bob"} 
}
