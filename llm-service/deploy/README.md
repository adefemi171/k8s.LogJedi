# LLM service deployment

## Using OpenAI in the cluster

1. **Create the Secret** (run once; replace with your key):
   ```bash
   kubectl -n logjedi create secret generic logjedi-llm-openai \
     --from-literal=LLM_API_KEY='sk-your-openai-key'
   ```

2. **Build and load the image** (from repo root):
   ```bash
   make docker-build
   # If using kind:
   kind load docker-image logjedi-llm-service:latest --name logjedi
   ```

3. **Deploy**:
   ```bash
   kubectl apply -f llm-service/deploy/
   ```

4. **Restart** so the new image is used:
   ```bash
   kubectl -n logjedi rollout restart deployment/llm-service
   ```

5. **Verify**: Port-forward and call analyze, or trigger a failing workload and check operator logs for "LLM analysis received" with a non-mock summary.

To switch back to mock, set `LLM_PROVIDER=mock` in the Deployment env and remove the `LLM_API_KEY` secretRef.
