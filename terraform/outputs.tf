output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.main.id
}

output "subnet_id" {
  description = "ID of the primary subnet"
  value       = aws_subnet.primary.id
}

output "cluster_endpoint" {
  description = "Endpoint of the EKS cluster"
  value       = aws_eks_cluster.microservices.endpoint
}

output "cluster_name" {
  description = "Name of the created EKS cluster"
  value       = aws_eks_cluster.microservices.name
}
