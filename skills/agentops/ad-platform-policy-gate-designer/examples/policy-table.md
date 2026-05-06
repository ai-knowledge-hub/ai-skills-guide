# Example Policy Gates

| Action | Auto-execute | Approval required | Deny when |
| --- | --- | --- | --- |
| Bid decrease | <= 5% and reversible | > 5% | Entity not scoped to workspace |
| Bid increase | <= 3% on low-spend entity | > 3% | Sensitive category without owner approval |
| Budget increase | Never by default | Any increase | > 20% daily delta |
| Targeting narrowing | <= approved taxonomy | Broad taxonomy change | Cross-client audience use |
| Targeting broadening | Never by default | Any broadening | Regulated or sensitive market without specialist approval |
