# Steps to start:
## 1. Apply everything  


Apply every file in every dir via


```bash
kubectl apply -f <file>
```

## 2. Run port forwarding

```bash
kubectl port-forward --address 0.0.0.0 deployment/nginx 30080:80
```
