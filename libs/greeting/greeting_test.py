import unittest
from libs.greeting.greeting import greet


class GreetingTest(unittest.TestCase):
    def test_greet(self):
        self.assertEqual(greet("kris"), "hello, kris")

    def test_greet_everyone_in_testdata(self):
        with open("libs/greeting/testdata/names.txt") as f:
            names = [line.strip() for line in f if line.strip()]
        for n in names:
            self.assertTrue(greet(n).startswith("hello, "))


if __name__ == "__main__":
    unittest.main()
