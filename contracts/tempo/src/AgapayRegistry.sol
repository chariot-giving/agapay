// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title AgapayRegistry
/// @notice On-chain nonprofit registry for the Agapay payment network.
/// Issuers (service providers) register organizations after KYB/KYC verification.
/// Organizations are keyed by EIN to prevent duplicates. This contract is the
/// Solidity equivalent of the Anchor program at programs/agapay-registry/.
contract AgapayRegistry {
    // -----------------------------------------------------------------------
    // Data Structures
    // -----------------------------------------------------------------------

    struct Issuer {
        address authority;
        string did;
        bool active;
        uint64 createdAt;
    }

    struct Organization {
        string ein;
        string name;
        string domain;
        string didUri;
        address issuer;
        bytes32 vcHash;
        address paymentAddress;
        bool active;
        uint64 createdAt;
        uint64 updatedAt;
    }

    // -----------------------------------------------------------------------
    // State
    // -----------------------------------------------------------------------

    address public owner;

    /// @dev issuer authority address => Issuer
    mapping(address => Issuer) private issuers;

    /// @dev keccak256(ein) => Organization
    mapping(bytes32 => Organization) private organizations;

    // -----------------------------------------------------------------------
    // Events
    // -----------------------------------------------------------------------

    event IssuerRegistered(address indexed authority, string did);
    event IssuerDeactivated(address indexed authority);

    event OrganizationRegistered(
        string indexed einHash, string ein, string name, address indexed issuer, address paymentAddress
    );

    event OrganizationUpdated(string indexed einHash, string ein, address indexed issuer);

    event OrganizationDeactivated(string indexed einHash, string ein, address indexed issuer);

    // -----------------------------------------------------------------------
    // Errors
    // -----------------------------------------------------------------------

    error NotOwner();
    error IssuerAlreadyRegistered();
    error IssuerNotFound();
    error IssuerNotActive();
    error OrganizationAlreadyExists();
    error OrganizationNotFound();
    error NotIssuingIssuer();

    // -----------------------------------------------------------------------
    // Constructor
    // -----------------------------------------------------------------------

    constructor() {
        owner = msg.sender;
    }

    // -----------------------------------------------------------------------
    // Issuer Management (owner-gated)
    // -----------------------------------------------------------------------

    /// @notice Register a new issuer (service provider). Only the contract owner
    /// can call this, mirroring the Anchor program's authority constraint.
    function registerIssuer(address authority, string calldata did) external {
        if (msg.sender != owner) revert NotOwner();
        if (issuers[authority].createdAt != 0) revert IssuerAlreadyRegistered();

        issuers[authority] = Issuer({authority: authority, did: did, active: true, createdAt: uint64(block.timestamp)});

        emit IssuerRegistered(authority, did);
    }

    /// @notice Deactivate an issuer.
    function deactivateIssuer(address authority) external {
        if (msg.sender != owner) revert NotOwner();
        if (issuers[authority].createdAt == 0) revert IssuerNotFound();

        issuers[authority].active = false;
        emit IssuerDeactivated(authority);
    }

    // -----------------------------------------------------------------------
    // Organization Management (issuer-gated)
    // -----------------------------------------------------------------------

    /// @notice Register a new organization. Only active issuers can call this.
    function registerOrganization(
        string calldata ein,
        string calldata name,
        string calldata domain,
        string calldata didUri,
        bytes32 vcHash,
        address paymentAddress
    ) external {
        _requireActiveIssuer(msg.sender);

        bytes32 key = keccak256(bytes(ein));
        if (organizations[key].createdAt != 0) revert OrganizationAlreadyExists();

        organizations[key] = Organization({
            ein: ein,
            name: name,
            domain: domain,
            didUri: didUri,
            issuer: msg.sender,
            vcHash: vcHash,
            paymentAddress: paymentAddress,
            active: true,
            createdAt: uint64(block.timestamp),
            updatedAt: uint64(block.timestamp)
        });

        emit OrganizationRegistered(ein, ein, name, msg.sender, paymentAddress);
    }

    /// @notice Update an existing organization. Only the original issuer can update.
    function updateOrganization(
        string calldata ein,
        string calldata name,
        string calldata domain,
        string calldata didUri,
        bytes32 vcHash,
        address paymentAddress
    ) external {
        _requireActiveIssuer(msg.sender);

        bytes32 key = keccak256(bytes(ein));
        Organization storage org = organizations[key];
        if (org.createdAt == 0) revert OrganizationNotFound();
        if (org.issuer != msg.sender) revert NotIssuingIssuer();

        org.name = name;
        org.domain = domain;
        org.didUri = didUri;
        org.vcHash = vcHash;
        org.paymentAddress = paymentAddress;
        org.updatedAt = uint64(block.timestamp);

        emit OrganizationUpdated(ein, ein, msg.sender);
    }

    /// @notice Deactivate an organization. Only the original issuer can deactivate.
    function deactivateOrganization(string calldata ein) external {
        _requireActiveIssuer(msg.sender);

        bytes32 key = keccak256(bytes(ein));
        Organization storage org = organizations[key];
        if (org.createdAt == 0) revert OrganizationNotFound();
        if (org.issuer != msg.sender) revert NotIssuingIssuer();

        org.active = false;
        org.updatedAt = uint64(block.timestamp);

        emit OrganizationDeactivated(ein, ein, msg.sender);
    }

    // -----------------------------------------------------------------------
    // Read Functions
    // -----------------------------------------------------------------------

    /// @notice Get an organization by EIN.
    function getOrganization(string calldata ein) external view returns (Organization memory) {
        bytes32 key = keccak256(bytes(ein));
        Organization memory org = organizations[key];
        if (org.createdAt == 0) revert OrganizationNotFound();
        return org;
    }

    /// @notice Get an issuer by authority address.
    function getIssuer(address authority) external view returns (Issuer memory) {
        Issuer memory iss = issuers[authority];
        if (iss.createdAt == 0) revert IssuerNotFound();
        return iss;
    }

    // -----------------------------------------------------------------------
    // Internal
    // -----------------------------------------------------------------------

    function _requireActiveIssuer(address authority) internal view {
        if (issuers[authority].createdAt == 0) revert IssuerNotFound();
        if (!issuers[authority].active) revert IssuerNotActive();
    }
}
