# KBT-CUY Powerbank Rental ![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white) ![SQLite](https://img.shields.io/badge/sqlite-%2307405e.svg?style=for-the-badge&logo=sqlite&logoColor=white)

KBT-CUY is a web-based power bank rental application built with Go. It allows users to find nearby power bank stations, rent a power bank, and return it to any available station. The system uses a local SQLite database and integrates with Midtrans for payment processing.

## ✨ Project Overview

This application provides a seamless solution for renting power banks on the go. Key features include:

-   **User Authentication**: Secure user registration and login functionality.
-   **Interactive Map**: A map view to locate all available power bank stations.
-   **Rental & Return Flow**: A simple process to rent a power bank from one station and return it to another.
-   **Payment Integration**: Utilizes Midtrans for handling payment transactions.
-   **Real-time Status**: View the availability of power banks at each station.

## 🛠️ Tools & Technologies

-   **Backend**: [Go](https://golang.org/)
-   **Web Framework**: [Gin](https://gin-gonic.com/)
-   **Database**: [SQLite](https://www.sqlite.org/)
-   **ORM**: [GORM](https://gorm.io/)
-   **Payment Gateway**: [Midtrans](https://midtrans.com/)
-   **Frontend**: HTML Templates (server-side rendered with Gin)

## 🚀 How to Use

Follow these steps to get the project up and running on your local machine.

### 1. Prerequisites

-   [Go](https://golang.org/dl/) (version 1.20 or higher recommended)
-   A Midtrans account to get your Server Key and Client Key.

### 2. Installation & Setup

1.  **Clone the repository:**
    ```sh
    git clone <your-repository-url>
    cd kbt-cuy
    ```

2.  **Install dependencies:**
    The project uses Go Modules. The dependencies will be downloaded automatically when you run the application.

3.  **Set up environment variables:**
    Create a `.env` file in the root of the `kbt-cuy` directory and add your Midtrans API keys:
    ```env
    MIDTRANS_SERVER_KEY="YOUR_MIDTRANS_SERVER_KEY"
    MIDTRANS_CLIENT_KEY="YOUR_MIDTRANS_CLIENT_KEY"
    ```

### 3. Running the Application

1.  **Run the main application:**
    ```sh
    go run main.go
    ```

2.  **Access the application:**
    Open your web browser and navigate to `http://localhost:8085`.

The application will start, and the database (`powerbank.db`) will be created automatically with some initial seed data for testing purposes.