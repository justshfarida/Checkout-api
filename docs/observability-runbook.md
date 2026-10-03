# Checkout API alert runbook

## Initial checks
- Open the Golden Signals dashboard.
- Check the alert's time window and affected environment.
- Check Kubernetes pods:
  kubectl get pods -n workshop
- Check backend logs:
  kubectl logs deployment/checkout-api -n workshop --tail=100

## High error rate
- Find HTTP 5xx requests in Datadog APM.
- Inspect the failing endpoint and its correlated logs.
- Check database connectivity and external API failures.
- During the homework test, /telemetry-test-error deliberately returns 500.

## High latency
- Find slow requests in APM.
- Inspect child spans to locate slow operations.
- Check database queries, external API calls, and CPU/memory usage.

## Abnormal traffic
- Check whether the frontend and backend are reachable.
- Check the backend Service, Ingress, and minikube tunnel.
- Determine whether the traffic change is expected.

## High saturation
- Check backend CPU and memory usage against container limits.
- Investigate increased load, memory growth, and container restarts.
- Reduce unnecessary work or adjust resources when appropriate.

## Verify recovery
- Apply the fix and generate successful requests.
- Confirm the monitored value returns below its threshold.
- Confirm Datadog recovers and Slack receives a recovery notification.