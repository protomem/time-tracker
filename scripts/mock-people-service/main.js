import express from "express";
import db from "./db.js";

const app = express();
const PORT = process.env.PORT || 3000;

app.get("/info", (req, res) => {
  const passportSerie = parseInt(req.query.passportSerie, 10);
  const passportNumber = parseInt(req.query.passportNumber, 10);

  if (isNaN(passportSerie) || isNaN(passportNumber)) {
    return res.status(400).json({ message: "Invalid query parameters" });
  }

  try {
    const people = db.peopleData.find((person) => {
      const parts = person.passport.split(" ");
      return (
        passportSerie === parseInt(parts[0], 10) &&
        passportNumber === parseInt(parts[1], 10)
      );
    });

    if (!people) throw new Error("Person not found");

    return res.status(200).json(people);
  } catch (error) {
    console.error(error);
    return res.status(500).json({ message: "Internal server error" });
  }
});

app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
