# GitHub Secrets Setup Guide

This guide will help you configure the required secrets for the CI/CD pipeline.

## Prerequisites

- GitHub repository access with admin permissions
- OCI account with API key configured
- OCI compute instance created and running
- SSH access to OCI instance

## Step-by-Step Setup

### Step 1: Generate OCI API Key (if not already done)

```bash
# Create OCI directory
mkdir -p ~/.oci

# Generate API key pair
openssl genrsa -out ~/.oci/oci_api_key.pem 2048

# Generate public key
openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem

# Get fingerprint (save this)
openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c

# Display public key to add to OCI
cat ~/.oci/oci_api_key_public.pem
```

**Add to OCI:**
1. Login to OCI Console
2. Click profile icon > User Settings
3. Click "API Keys" > "Add API Key"
4. Paste public key content
5. Save

### Step 2: Get OCI OCIDs

**User OCID:**
1. OCI Console > Profile Icon > User Settings
2. Copy OCID (starts with `ocid1.user.oc1..`)

**Tenancy OCID:**
1. OCI Console > Profile Icon > Tenancy
2. Copy OCID (starts with `ocid1.tenancy.oc1..`)

**Region:**
- Example: `us-phoenix-1`, `us-ashburn-1`, `uk-london-1`
- Find in OCI Console top-right

### Step 3: Generate SSH Key for OCI Instance

```bash
# Generate SSH key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/oci_todolist_key -N ""

# Display private key (for GitHub secret)
cat ~/.ssh/oci_todolist_key

# Display public key (for OCI instance)
cat ~/.ssh/oci_todolist_key.pub
```

**Add to OCI Instance:**

Option A - During instance creation:
- Paste public key content when prompted

Option B - Existing instance:
```bash
# SSH to instance with existing key
ssh ubuntu@<INSTANCE_IP>

# Add new public key
echo "<PUBLIC_KEY_CONTENT>" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

### Step 4: Add Secrets to GitHub

1. Go to your GitHub repository
2. Click `Settings` > `Secrets and variables` > `Actions`
3. Click `New repository secret`
4. Add each secret below:

#### OCI_CLI_USER
```
Name: OCI_CLI_USER
Value: ocid1.user.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

#### OCI_CLI_TENANCY
```
Name: OCI_CLI_TENANCY
Value: ocid1.tenancy.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

#### OCI_CLI_FINGERPRINT
```
Name: OCI_CLI_FINGERPRINT
Value: aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
```

#### OCI_CLI_KEY_CONTENT
```
Name: OCI_CLI_KEY_CONTENT
Value:
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
(entire private key content from ~/.oci/oci_api_key.pem)
...
-----END RSA PRIVATE KEY-----
```

**Important:** Include the `BEGIN` and `END` lines

#### OCI_CLI_REGION
```
Name: OCI_CLI_REGION
Value: us-phoenix-1
```

#### OCI_INSTANCE_IP
```
Name: OCI_INSTANCE_IP
Value: 132.145.xxx.xxx
```

Get from: OCI Console > Compute > Instances > Your Instance > Public IP

#### OCI_USER
```
Name: OCI_USER
Value: ubuntu
```

(or `opc` for Oracle Linux)

#### OCI_SSH_PRIVATE_KEY
```
Name: OCI_SSH_PRIVATE_KEY
Value:
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAACFwAAAAdzc2gtcn
(entire private key content from ~/.ssh/oci_todolist_key)
...
-----END OPENSSH PRIVATE KEY-----
```

**Important:** Include the `BEGIN` and `END` lines

### Step 5: Optional - Codecov Token

If you want code coverage reports:

1. Go to [codecov.io](https://codecov.io)
2. Sign in with GitHub
3. Add your repository
4. Copy the token
5. Add to GitHub secrets:

```
Name: CODECOV_TOKEN
Value: <your-codecov-token>
```

## Verification Checklist

- [ ] OCI API key added to OCI Console
- [ ] SSH public key added to OCI instance
- [ ] Can SSH to instance: `ssh -i ~/.ssh/oci_todolist_key ubuntu@<IP>`
- [ ] All 8 required secrets added to GitHub
- [ ] Secret values have no extra spaces or newlines
- [ ] Private keys include BEGIN/END markers
- [ ] OCIDs are complete (no truncation)

## Testing the Setup

### Test 1: SSH Connection

```bash
ssh -i ~/.ssh/oci_todolist_key ubuntu@<OCI_INSTANCE_IP>
```

Should connect without password prompt.

### Test 2: OCI CLI (Local)

```bash
# Install OCI CLI
brew install oci-cli  # macOS
# or
pip install oci-cli    # Python

# Test configuration
oci iam region list --config-file ~/.oci/config
```

### Test 3: Trigger Workflow

1. Make a small change (e.g., update README)
2. Commit and push to `develop` branch
3. Go to Actions tab in GitHub
4. Watch workflow run
5. Check for any errors

### Test 4: Manual Workflow Trigger

1. Go to Actions tab
2. Select "CI/CD Pipeline"
3. Click "Run workflow"
4. Select `develop` branch
5. Click "Run workflow"
6. Monitor execution

## Common Issues

### Issue: "Authentication failed" in workflow

**Fix:**
- Verify `OCI_CLI_KEY_CONTENT` is complete with BEGIN/END lines
- Check fingerprint matches: `openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c`
- Ensure no extra spaces in secret values

### Issue: "SSH connection refused"

**Fix:**
- Verify security list allows SSH (port 22) from GitHub IPs
- Check `OCI_INSTANCE_IP` is correct public IP
- Ensure `OCI_SSH_PRIVATE_KEY` is complete
- Test SSH manually first

### Issue: "Permission denied (publickey)"

**Fix:**
- Verify SSH public key is in `~/.ssh/authorized_keys` on instance
- Check `OCI_USER` is correct (ubuntu vs opc)
- Ensure private key format is correct (OpenSSH format)

### Issue: Workflow fails at deployment step

**Fix:**
- Check OCI instance is running
- Verify systemd service can be created
- Check disk space: `df -h`
- Review deployment logs in Actions

## Security Best Practices

✅ **Never commit secrets to git**
✅ **Rotate API keys every 90 days**
✅ **Use separate SSH keys for CI/CD**
✅ **Limit OCI user permissions** (principle of least privilege)
✅ **Enable 2FA on GitHub account**
✅ **Review secret access logs** periodically
✅ **Use environment protection rules** for production

## Updating Secrets

To update a secret:
1. Go to Settings > Secrets and variables > Actions
2. Find the secret
3. Click "Update"
4. Enter new value
5. Click "Update secret"

**Note:** Workflows don't automatically re-run when secrets change. You must trigger a new run.

## Removing Old Keys

When rotating keys, don't forget to:
1. Remove old API key from OCI Console
2. Remove old SSH key from OCI instance
3. Update secrets in GitHub
4. Test new configuration

## Next Steps

After setting up secrets:
1. ✅ Review [Workflow README](.github/workflows/README.md)
2. ✅ Push to `develop` branch to test
3. ✅ Monitor first workflow run
4. ✅ Fix any issues
5. ✅ Push to `main` for production deployment

## Support

If you encounter issues:
1. Check workflow logs in Actions tab
2. Verify all secrets are configured correctly
3. Test SSH and OCI CLI access locally
4. Review OCI instance security lists
5. Check OCI instance system logs

---

**Last Updated:** 2025-11-12
