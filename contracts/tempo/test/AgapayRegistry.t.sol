// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {AgapayRegistry} from "../src/AgapayRegistry.sol";

contract AgapayRegistryTest is Test {
    AgapayRegistry public registry;

    address owner = address(this);
    address issuerAuthority = makeAddr("issuer");
    address otherUser = makeAddr("other");
    address paymentAddr = makeAddr("payment");

    string constant DID = "did:web:givechariot.com";
    string constant EIN = "530196605";
    string constant ORG_NAME = "American Red Cross";
    string constant DOMAIN = "redcross.org";
    string constant DID_URI = "did:web:agapay.redcross.org";
    bytes32 constant VC_HASH = keccak256("test-vc-hash");

    function setUp() public {
        registry = new AgapayRegistry();
    }

    // -----------------------------------------------------------------------
    // Issuer Registration
    // -----------------------------------------------------------------------

    function test_RegisterIssuer() public {
        registry.registerIssuer(issuerAuthority, DID);

        AgapayRegistry.Issuer memory iss = registry.getIssuer(issuerAuthority);
        assertEq(iss.authority, issuerAuthority);
        assertEq(iss.did, DID);
        assertTrue(iss.active);
        assertGt(iss.createdAt, 0);
    }

    function test_RegisterIssuer_EmitsEvent() public {
        vm.expectEmit(true, false, false, true);
        emit AgapayRegistry.IssuerRegistered(issuerAuthority, DID);

        registry.registerIssuer(issuerAuthority, DID);
    }

    function test_RegisterIssuer_RevertsIfNotOwner() public {
        vm.prank(otherUser);
        vm.expectRevert(AgapayRegistry.NotOwner.selector);
        registry.registerIssuer(issuerAuthority, DID);
    }

    function test_RegisterIssuer_RevertsIfAlreadyRegistered() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.expectRevert(AgapayRegistry.IssuerAlreadyRegistered.selector);
        registry.registerIssuer(issuerAuthority, DID);
    }

    function test_DeactivateIssuer() public {
        registry.registerIssuer(issuerAuthority, DID);
        registry.deactivateIssuer(issuerAuthority);

        AgapayRegistry.Issuer memory iss = registry.getIssuer(issuerAuthority);
        assertFalse(iss.active);
    }

    function test_DeactivateIssuer_RevertsIfNotOwner() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.prank(otherUser);
        vm.expectRevert(AgapayRegistry.NotOwner.selector);
        registry.deactivateIssuer(issuerAuthority);
    }

    function test_DeactivateIssuer_RevertsIfNotFound() public {
        vm.expectRevert(AgapayRegistry.IssuerNotFound.selector);
        registry.deactivateIssuer(issuerAuthority);
    }

    // -----------------------------------------------------------------------
    // Organization Registration
    // -----------------------------------------------------------------------

    function test_RegisterOrganization() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.prank(issuerAuthority);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);

        AgapayRegistry.Organization memory org = registry.getOrganization(EIN);
        assertEq(org.ein, EIN);
        assertEq(org.name, ORG_NAME);
        assertEq(org.domain, DOMAIN);
        assertEq(org.didUri, DID_URI);
        assertEq(org.issuer, issuerAuthority);
        assertEq(org.vcHash, VC_HASH);
        assertEq(org.paymentAddress, paymentAddr);
        assertTrue(org.active);
        assertGt(org.createdAt, 0);
        assertEq(org.createdAt, org.updatedAt);
    }

    function test_RegisterOrganization_RevertsIfNotIssuer() public {
        vm.prank(otherUser);
        vm.expectRevert(AgapayRegistry.IssuerNotFound.selector);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);
    }

    function test_RegisterOrganization_RevertsIfIssuerInactive() public {
        registry.registerIssuer(issuerAuthority, DID);
        registry.deactivateIssuer(issuerAuthority);

        vm.prank(issuerAuthority);
        vm.expectRevert(AgapayRegistry.IssuerNotActive.selector);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);
    }

    function test_RegisterOrganization_RevertsIfDuplicateEIN() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.startPrank(issuerAuthority);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);

        vm.expectRevert(AgapayRegistry.OrganizationAlreadyExists.selector);
        registry.registerOrganization(EIN, "Other Org", DOMAIN, DID_URI, VC_HASH, paymentAddr);
        vm.stopPrank();
    }

    // -----------------------------------------------------------------------
    // Organization Update
    // -----------------------------------------------------------------------

    function test_UpdateOrganization() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.startPrank(issuerAuthority);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);

        address newPayment = makeAddr("newPayment");
        bytes32 newHash = keccak256("updated-vc-hash");
        vm.warp(block.timestamp + 100);

        registry.updateOrganization(EIN, "Updated Name", "new.domain.org", DID_URI, newHash, newPayment);
        vm.stopPrank();

        AgapayRegistry.Organization memory org = registry.getOrganization(EIN);
        assertEq(org.name, "Updated Name");
        assertEq(org.domain, "new.domain.org");
        assertEq(org.vcHash, newHash);
        assertEq(org.paymentAddress, newPayment);
        assertGt(org.updatedAt, org.createdAt);
    }

    function test_UpdateOrganization_RevertsIfWrongIssuer() public {
        address issuer2 = makeAddr("issuer2");
        registry.registerIssuer(issuerAuthority, DID);
        registry.registerIssuer(issuer2, "did:web:other.com");

        vm.prank(issuerAuthority);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);

        vm.prank(issuer2);
        vm.expectRevert(AgapayRegistry.NotIssuingIssuer.selector);
        registry.updateOrganization(EIN, "Hacked", DOMAIN, DID_URI, VC_HASH, paymentAddr);
    }

    // -----------------------------------------------------------------------
    // Organization Deactivation
    // -----------------------------------------------------------------------

    function test_DeactivateOrganization() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.prank(issuerAuthority);
        registry.registerOrganization(EIN, ORG_NAME, DOMAIN, DID_URI, VC_HASH, paymentAddr);

        vm.prank(issuerAuthority);
        registry.deactivateOrganization(EIN);

        AgapayRegistry.Organization memory org = registry.getOrganization(EIN);
        assertFalse(org.active);
    }

    function test_DeactivateOrganization_RevertsIfNotFound() public {
        registry.registerIssuer(issuerAuthority, DID);

        vm.prank(issuerAuthority);
        vm.expectRevert(AgapayRegistry.OrganizationNotFound.selector);
        registry.deactivateOrganization("999999999");
    }

    // -----------------------------------------------------------------------
    // Read Functions
    // -----------------------------------------------------------------------

    function test_GetIssuer_RevertsIfNotFound() public {
        vm.expectRevert(AgapayRegistry.IssuerNotFound.selector);
        registry.getIssuer(otherUser);
    }

    function test_GetOrganization_RevertsIfNotFound() public {
        vm.expectRevert(AgapayRegistry.OrganizationNotFound.selector);
        registry.getOrganization("000000000");
    }
}
