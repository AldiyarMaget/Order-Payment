terraform {
  required_version = ">= 1.0.0"

  # Local backend placeholder
  # To switch to S3 remote backend, uncomment the block below and remove local backend if present
  backend "local" {
    path = "terraform.tfstate"
  }

  /*
  backend "s3" {
    bucket         = "my-terraform-state-bucket"
    key            = "microservices/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-lock-table"
  }
  */

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Example VPC for the cluster
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name        = "${var.cluster_name}-vpc"
    Environment = var.environment
  }
}

# Subnet for the cluster
resource "aws_subnet" "primary" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true

  tags = {
    Name        = "${var.cluster_name}-subnet"
    Environment = var.environment
  }
}

# Mock EKS Cluster definition
resource "aws_eks_cluster" "microservices" {
  name     = var.cluster_name
  role_arn = "arn:aws:iam::123456789012:role/eks-cluster-role-mock"

  vpc_config {
    subnet_ids = [aws_subnet.primary.id]
  }

  tags = {
    Environment = var.environment
  }

  # Lifecycle rule to ignore missing role locally without valid credentials
  lifecycle {
    ignore_changes = [role_arn]
  }
}
