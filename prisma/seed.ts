import { address_status, bank_address_status, PrismaClient } from "../dist";
import * as dotenv from "dotenv";
import * as fs from "fs";
import * as path from "path";
dotenv.config();

const prisma = new PrismaClient();

async function seedUsers() {
  const user = await prisma.user.upsert({
    where: {
      id: 0,
    },
    create: {
      id: 0,
      email: "",
    },
    update: {},
  });

  console.log("Created user", user);
}

async function seedOrganizations() {
  const cgn = await prisma.organization.upsert({
    where: {
      id: "org_1",
    },
    create: {
      id: "org_1",
      legal_name: "Chariot Giving Network",
      ein: "931372175",
      address: {
        create: {
          line1: "850 7th Ave",
          line2: "Suite 600",
          city: "New York",
          state: "NY",
          postalCode: "10019",
          status: address_status.active,
        },
      },
    },
    update: {},
  });

  console.log("Created organization", cgn);
}

async function seedRecipients() {
  const cgn = await prisma.recipient.upsert({
    where: {
      id: "e8ff4be1-4603-4bb8-95f3-953c7b95882b",
    },
    create: {
      id: "e8ff4be1-4603-4bb8-95f3-953c7b95882b",
      name: "Chariot Giving Network",
      primary: true,
      organization: {
        connect: {
          id: "org_1",
        },
      },
      bankAddress: {
        create: {
          account_number: "00100000132239778",
          routing_number: "028000121",
          status: bank_address_status.active,
        },
      },
      mailingAddress: {
        create: {
          line1: "PO Box 2235",
          city: "New York",
          state: "NY",
          postalCode: "10101",
          status: address_status.active,
        },
      },
    },
    update: {},
  });

  console.log("Created recipient", cgn);
}

async function main() {
  console.time("seed");

  console.log("Seeding users...");
  await seedUsers();
  console.timeLog("seed", "Successfully seeded users");

  console.log("Seeding organizations...");
  await seedOrganizations();
  console.timeLog("seed", "Successfully seeded organizations");

  console.log("Seeding recipients...");
  await seedRecipients();
  console.timeLog("seed", "Successfully seeded recipients");

  console.timeLog("Finished running seed");
  console.timeEnd("seed");
}

main()
  .then(async () => {
    await prisma.$disconnect();
  })
  .catch(async (e) => {
    console.error(e);
    await prisma.$disconnect();
    process.exit(1);
  });
