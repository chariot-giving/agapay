use anchor_lang::prelude::*;

declare_id!("HBtGrueW38KAi6CAfVG22MJopasM47GY6DFvq2W1U98j");

/// Maximum length for string fields stored on-chain.
const MAX_DID_LEN: usize = 128;
const MAX_NAME_LEN: usize = 128;
const MAX_DOMAIN_LEN: usize = 128;
const EIN_LEN: usize = 9;

#[program]
pub mod agapay_registry {
    use super::*;

    /// Register a new Service Provider as an authorized issuer.
    /// Only the program authority (upgrade authority) can call this.
    pub fn register_issuer(ctx: Context<RegisterIssuer>, did: String) -> Result<()> {
        require!(did.len() <= MAX_DID_LEN, AgapayError::FieldTooLong);
        require!(did.starts_with("did:"), AgapayError::InvalidDid);

        let issuer = &mut ctx.accounts.issuer_account;
        issuer.authority = ctx.accounts.issuer_authority.key();
        issuer.did = did;
        issuer.active = true;
        issuer.created_at = Clock::get()?.unix_timestamp;

        msg!(
            "Issuer registered: {}",
            ctx.accounts.issuer_authority.key()
        );
        Ok(())
    }

    /// Register a new nonprofit organization in the on-chain registry.
    /// Only a registered, active issuer can call this.
    pub fn register_organization(
        ctx: Context<RegisterOrganization>,
        ein: String,
        name: String,
        domain: String,
        did_uri: String,
        vc_hash: [u8; 32],
        usdc_address: Pubkey,
    ) -> Result<()> {
        require!(ein.len() == EIN_LEN, AgapayError::InvalidEin);
        require!(
            ein.chars().all(|c| c.is_ascii_digit()),
            AgapayError::InvalidEin
        );
        require!(name.len() <= MAX_NAME_LEN, AgapayError::FieldTooLong);
        require!(domain.len() <= MAX_DOMAIN_LEN, AgapayError::FieldTooLong);
        require!(did_uri.len() <= MAX_DID_LEN, AgapayError::FieldTooLong);
        require!(did_uri.starts_with("did:"), AgapayError::InvalidDid);

        let issuer = &ctx.accounts.issuer_account;
        require!(issuer.active, AgapayError::IssuerInactive);
        require!(
            issuer.authority == ctx.accounts.issuer_authority.key(),
            AgapayError::Unauthorized
        );

        let now = Clock::get()?.unix_timestamp;
        let org = &mut ctx.accounts.organization_account;
        org.ein = ein;
        org.name = name;
        org.domain = domain;
        org.did_uri = did_uri;
        org.issuer = ctx.accounts.issuer_authority.key();
        org.vc_hash = vc_hash;
        org.usdc_address = usdc_address;
        org.active = true;
        org.created_at = now;
        org.updated_at = now;

        msg!("Organization registered: EIN {}", org.ein);
        Ok(())
    }

    /// Update an existing organization record.
    /// Only the issuer who originally registered the organization can update it.
    pub fn update_organization(
        ctx: Context<UpdateOrganization>,
        name: Option<String>,
        domain: Option<String>,
        did_uri: Option<String>,
        vc_hash: Option<[u8; 32]>,
        usdc_address: Option<Pubkey>,
    ) -> Result<()> {
        let issuer = &ctx.accounts.issuer_account;
        require!(issuer.active, AgapayError::IssuerInactive);
        require!(
            issuer.authority == ctx.accounts.issuer_authority.key(),
            AgapayError::Unauthorized
        );

        let org = &mut ctx.accounts.organization_account;
        require!(
            org.issuer == ctx.accounts.issuer_authority.key(),
            AgapayError::Unauthorized
        );

        if let Some(n) = name {
            require!(n.len() <= MAX_NAME_LEN, AgapayError::FieldTooLong);
            org.name = n;
        }
        if let Some(d) = domain {
            require!(d.len() <= MAX_DOMAIN_LEN, AgapayError::FieldTooLong);
            org.domain = d;
        }
        if let Some(d) = did_uri {
            require!(d.len() <= MAX_DID_LEN, AgapayError::FieldTooLong);
            require!(d.starts_with("did:"), AgapayError::InvalidDid);
            org.did_uri = d;
        }
        if let Some(h) = vc_hash {
            org.vc_hash = h;
        }
        if let Some(a) = usdc_address {
            org.usdc_address = a;
        }

        org.updated_at = Clock::get()?.unix_timestamp;

        msg!("Organization updated: EIN {}", org.ein);
        Ok(())
    }

    /// Deactivate an organization, marking it as inactive in the registry.
    /// Only the issuer who registered the organization can deactivate it.
    pub fn deactivate_organization(ctx: Context<DeactivateOrganization>) -> Result<()> {
        let issuer = &ctx.accounts.issuer_account;
        require!(issuer.active, AgapayError::IssuerInactive);
        require!(
            issuer.authority == ctx.accounts.issuer_authority.key(),
            AgapayError::Unauthorized
        );

        let org = &mut ctx.accounts.organization_account;
        require!(
            org.issuer == ctx.accounts.issuer_authority.key(),
            AgapayError::Unauthorized
        );

        org.active = false;
        org.updated_at = Clock::get()?.unix_timestamp;

        msg!("Organization deactivated: EIN {}", org.ein);
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Account structs
// ---------------------------------------------------------------------------

#[account]
#[derive(InitSpace)]
pub struct IssuerAccount {
    /// The signing authority for this issuer.
    pub authority: Pubkey,
    /// DID URI of the issuer (e.g., "did:web:givechariot.com").
    #[max_len(MAX_DID_LEN)]
    pub did: String,
    /// Whether this issuer is currently authorized to write.
    pub active: bool,
    /// Unix timestamp of registration.
    pub created_at: i64,
}

#[account]
#[derive(InitSpace)]
pub struct OrganizationAccount {
    /// Employer Identification Number (9 digits).
    #[max_len(EIN_LEN)]
    pub ein: String,
    /// Organization name.
    #[max_len(MAX_NAME_LEN)]
    pub name: String,
    /// Verified web domain.
    #[max_len(MAX_DOMAIN_LEN)]
    pub domain: String,
    /// Organization's DID URI.
    #[max_len(MAX_DID_LEN)]
    pub did_uri: String,
    /// Pubkey of the issuer who registered this organization.
    pub issuer: Pubkey,
    /// SHA-256 hash of the issued Verifiable Credentials.
    pub vc_hash: [u8; 32],
    /// USDC SPL token account for receiving payments.
    pub usdc_address: Pubkey,
    /// Whether this organization is currently active.
    pub active: bool,
    /// Unix timestamp of creation.
    pub created_at: i64,
    /// Unix timestamp of last update.
    pub updated_at: i64,
}

// ---------------------------------------------------------------------------
// Instruction account contexts
// ---------------------------------------------------------------------------

#[derive(Accounts)]
pub struct RegisterIssuer<'info> {
    /// Program authority (must be the upgrade authority / deployer).
    #[account(mut)]
    pub authority: Signer<'info>,

    /// The pubkey that will act as the issuer's signing authority.
    /// CHECK: This is the issuer authority key; validated by the program logic.
    pub issuer_authority: UncheckedAccount<'info>,

    /// The IssuerAccount PDA to create, seeded by the issuer authority pubkey.
    #[account(
        init,
        payer = authority,
        space = 8 + IssuerAccount::INIT_SPACE,
        seeds = [b"issuer", issuer_authority.key().as_ref()],
        bump,
    )]
    pub issuer_account: Account<'info, IssuerAccount>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(ein: String)]
pub struct RegisterOrganization<'info> {
    /// The issuer's signing authority.
    #[account(mut)]
    pub issuer_authority: Signer<'info>,

    /// The issuer's IssuerAccount PDA.
    #[account(
        seeds = [b"issuer", issuer_authority.key().as_ref()],
        bump,
    )]
    pub issuer_account: Account<'info, IssuerAccount>,

    /// The OrganizationAccount PDA to create, seeded by the EIN.
    #[account(
        init,
        payer = issuer_authority,
        space = 8 + OrganizationAccount::INIT_SPACE,
        seeds = [b"organization", ein.as_bytes()],
        bump,
    )]
    pub organization_account: Account<'info, OrganizationAccount>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct UpdateOrganization<'info> {
    /// The issuer's signing authority.
    pub issuer_authority: Signer<'info>,

    /// The issuer's IssuerAccount PDA.
    #[account(
        seeds = [b"issuer", issuer_authority.key().as_ref()],
        bump,
    )]
    pub issuer_account: Account<'info, IssuerAccount>,

    /// The OrganizationAccount PDA to update.
    #[account(
        mut,
        seeds = [b"organization", organization_account.ein.as_bytes()],
        bump,
    )]
    pub organization_account: Account<'info, OrganizationAccount>,
}

#[derive(Accounts)]
pub struct DeactivateOrganization<'info> {
    /// The issuer's signing authority.
    pub issuer_authority: Signer<'info>,

    /// The issuer's IssuerAccount PDA.
    #[account(
        seeds = [b"issuer", issuer_authority.key().as_ref()],
        bump,
    )]
    pub issuer_account: Account<'info, IssuerAccount>,

    /// The OrganizationAccount PDA to deactivate.
    #[account(
        mut,
        seeds = [b"organization", organization_account.ein.as_bytes()],
        bump,
    )]
    pub organization_account: Account<'info, OrganizationAccount>,
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

#[error_code]
pub enum AgapayError {
    #[msg("Field exceeds maximum allowed length")]
    FieldTooLong,
    #[msg("EIN must be exactly 9 digits")]
    InvalidEin,
    #[msg("DID must start with 'did:'")]
    InvalidDid,
    #[msg("Issuer is not active")]
    IssuerInactive,
    #[msg("Unauthorized: signer does not match expected authority")]
    Unauthorized,
}
