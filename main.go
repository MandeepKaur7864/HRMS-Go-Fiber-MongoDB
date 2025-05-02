package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

var mg MongoInstance

const DBName = "fiber-hrms"

const mongoURI = "mongodb://localhost:27017/" + DBName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Age    float64 `json:"age"`
	Salary float64 `json:"salary"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	db := client.Database(DBName)

	mg = MongoInstance{
		Client: client,
		DB:     db,
	}

	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/getEmployees", func(c *fiber.Ctx) error {
		var employees []Employee
		query := bson.D{}
		cursor, err := mg.DB.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		if err = cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})

	app.Post("/createEmployee", func(c *fiber.Ctx) error {
		collection := mg.DB.Collection("employees")

		var employee Employee

		err := c.BodyParser(&employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		employee.ID = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}

		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})

	app.Put("/updateEmployee/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.SendStatus(400)
		}

		var employee Employee

		if err := c.BodyParser(&employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "email", Value: employee.Email},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.DB.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = id

		return c.Status(200).JSON(employee)
	})

	app.Delete("/deleteEmployee/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		deleteResult, err := mg.DB.Collection("employees").DeleteOne(c.Context(), query)
		if err != nil {
			return c.SendStatus(500)
		}

		if deleteResult.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("Record deleted successfully!")
	})

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
