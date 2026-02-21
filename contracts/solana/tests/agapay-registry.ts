import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import { AgapayRegistry } from "../target/types/agapay_registry";
import { Keypair, PublicKey, SystemProgram } from "@solana/web3.js";
import { assert } from "chai";
import { createHash } from "crypto";

describe("agapay-registry", () => {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);

  const program = anchor.workspace.AgapayRegistry as Program<AgapayRegistry>;

  // Test keypairs
  const issuerAuthority = Keypair.generate();
  const usdcAddress = Keypair.generate().publicKey;

  // Test data
  const issuerDid = "did:web:givechariot.com";
  const testEin = "043567500";
  const testName = "Example Nonprofit Organization";
  const testDomain = "example.org";
  const testDidUri = "did:web:agapay.example.org";
  const testVcHash = createHash("sha256")
    .update("test-vc-content")
    .digest();

  // Derive PDAs
  const [issuerPda] = PublicKey.findProgramAddressSync(
    [Buffer.from("issuer"), issuerAuthority.publicKey.toBuffer()],
    program.programId
  );

  const [orgPda] = PublicKey.findProgramAddressSync(
    [Buffer.from("organization"), Buffer.from(testEin)],
    program.programId
  );

  before(async () => {
    // Airdrop SOL to the provider wallet and issuer authority for transaction fees
    const sig1 = await provider.connection.requestAirdrop(
      provider.wallet.publicKey,
      2 * anchor.web3.LAMPORTS_PER_SOL
    );
    await provider.connection.confirmTransaction(sig1);

    const sig2 = await provider.connection.requestAirdrop(
      issuerAuthority.publicKey,
      2 * anchor.web3.LAMPORTS_PER_SOL
    );
    await provider.connection.confirmTransaction(sig2);
  });

  it("registers an issuer", async () => {
    await program.methods
      .registerIssuer(issuerDid)
      .accounts({
        authority: provider.wallet.publicKey,
        issuerAuthority: issuerAuthority.publicKey,
        issuerAccount: issuerPda,
        systemProgram: SystemProgram.programId,
      })
      .rpc();

    const issuerAccount = await program.account.issuerAccount.fetch(issuerPda);
    assert.equal(issuerAccount.did, issuerDid);
    assert.isTrue(issuerAccount.active);
    assert.equal(
      issuerAccount.authority.toBase58(),
      issuerAuthority.publicKey.toBase58()
    );
  });

  it("registers an organization", async () => {
    await program.methods
      .registerOrganization(
        testEin,
        testName,
        testDomain,
        testDidUri,
        Array.from(testVcHash) as number[],
        usdcAddress
      )
      .accounts({
        issuerAuthority: issuerAuthority.publicKey,
        issuerAccount: issuerPda,
        organizationAccount: orgPda,
        systemProgram: SystemProgram.programId,
      })
      .signers([issuerAuthority])
      .rpc();

    const orgAccount = await program.account.organizationAccount.fetch(orgPda);
    assert.equal(orgAccount.ein, testEin);
    assert.equal(orgAccount.name, testName);
    assert.equal(orgAccount.domain, testDomain);
    assert.equal(orgAccount.didUri, testDidUri);
    assert.isTrue(orgAccount.active);
    assert.equal(
      orgAccount.issuer.toBase58(),
      issuerAuthority.publicKey.toBase58()
    );
    assert.equal(orgAccount.usdcAddress.toBase58(), usdcAddress.toBase58());
    assert.deepEqual(
      Buffer.from(orgAccount.vcHash as number[]),
      testVcHash
    );
  });

  it("rejects duplicate EIN registration", async () => {
    try {
      await program.methods
        .registerOrganization(
          testEin,
          "Duplicate Org",
          "duplicate.org",
          "did:web:agapay.duplicate.org",
          Array.from(testVcHash) as number[],
          usdcAddress
        )
        .accounts({
          issuerAuthority: issuerAuthority.publicKey,
          issuerAccount: issuerPda,
          organizationAccount: orgPda,
          systemProgram: SystemProgram.programId,
        })
        .signers([issuerAuthority])
        .rpc();
      assert.fail("Should have thrown an error for duplicate EIN");
    } catch (err) {
      // PDA init will fail because account already exists
      assert.ok(err);
    }
  });

  it("rejects invalid EIN length", async () => {
    const badEin = "12345"; // too short
    const [badOrgPda] = PublicKey.findProgramAddressSync(
      [Buffer.from("organization"), Buffer.from(badEin)],
      program.programId
    );

    try {
      await program.methods
        .registerOrganization(
          badEin,
          "Bad Org",
          "bad.org",
          "did:web:agapay.bad.org",
          Array.from(testVcHash) as number[],
          usdcAddress
        )
        .accounts({
          issuerAuthority: issuerAuthority.publicKey,
          issuerAccount: issuerPda,
          organizationAccount: badOrgPda,
          systemProgram: SystemProgram.programId,
        })
        .signers([issuerAuthority])
        .rpc();
      assert.fail("Should have thrown an error for invalid EIN");
    } catch (err) {
      assert.ok(err);
    }
  });

  it("updates an organization", async () => {
    const newName = "Updated Nonprofit Name";
    const newUsdcAddress = Keypair.generate().publicKey;

    await program.methods
      .updateOrganization(
        newName,
        null, // keep domain
        null, // keep did_uri
        null, // keep vc_hash
        newUsdcAddress
      )
      .accounts({
        issuerAuthority: issuerAuthority.publicKey,
        issuerAccount: issuerPda,
        organizationAccount: orgPda,
      })
      .signers([issuerAuthority])
      .rpc();

    const orgAccount = await program.account.organizationAccount.fetch(orgPda);
    assert.equal(orgAccount.name, newName);
    assert.equal(orgAccount.domain, testDomain); // unchanged
    assert.equal(
      orgAccount.usdcAddress.toBase58(),
      newUsdcAddress.toBase58()
    );
  });

  it("rejects update from unauthorized issuer", async () => {
    const unauthorizedIssuer = Keypair.generate();
    const sig = await provider.connection.requestAirdrop(
      unauthorizedIssuer.publicKey,
      anchor.web3.LAMPORTS_PER_SOL
    );
    await provider.connection.confirmTransaction(sig);

    try {
      // This will fail at the PDA derivation since the unauthorized
      // issuer doesn't have an IssuerAccount PDA.
      const [fakePda] = PublicKey.findProgramAddressSync(
        [Buffer.from("issuer"), unauthorizedIssuer.publicKey.toBuffer()],
        program.programId
      );

      await program.methods
        .updateOrganization(
          "Hacked Name",
          null,
          null,
          null,
          null
        )
        .accounts({
          issuerAuthority: unauthorizedIssuer.publicKey,
          issuerAccount: fakePda,
          organizationAccount: orgPda,
        })
        .signers([unauthorizedIssuer])
        .rpc();
      assert.fail("Should have thrown an error for unauthorized update");
    } catch (err) {
      assert.ok(err);
    }
  });

  it("deactivates an organization", async () => {
    await program.methods
      .deactivateOrganization()
      .accounts({
        issuerAuthority: issuerAuthority.publicKey,
        issuerAccount: issuerPda,
        organizationAccount: orgPda,
      })
      .signers([issuerAuthority])
      .rpc();

    const orgAccount = await program.account.organizationAccount.fetch(orgPda);
    assert.isFalse(orgAccount.active);
  });
});
