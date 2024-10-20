package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var (
	rnd *rand.Rand = rand.New(rand.NewSource(42))

	OPINION_ADJECTIVES = []string{
		"adorable", "adventurous", "aggressive", "agreeable", "alert", "alive", "amused", "angry", "annoyed", "annoying", "anxious",
		"arrogant", "ashamed", "attractive", "average", "awful", "bad", "beautiful", "better", "bewildered", "black", "bloody", "blue",
		"blue-eyed", "blushing", "bored", "brainy", "brave", "breakable", "bright", "busy", "calm", "careful", "cautious", "charming",
		"cheerful", "clean", "clear", "clever", "cloudy", "clumsy", "colorful", "combative", "comfortable", "concerned", "condemned",
		"confused", "cooperative", "courageous", "crazy", "creepy", "crowded", "cruel", "curious", "cute", "dangerous", "dark", "dead",
		"defeated", "defiant", "delightful", "depressed", "determined", "different", "difficult", "disgusted", "distinct", "disturbed",
		"dizzy", "doubtful", "drab", "dull", "eager", "	easy", "elated", "elegant", "embarrassed", "enchanting", "encouraging",
		"energetic", "enthusiastic", "envious", "evil", "excited", "expensive", "exuberant", "fair", "faithful", "famous", "fancy",
		"fantastic", "fierce", "filthy", "fine", "foolish", "fragile", "frail", "frantic", "friendly", "frightened", "funny", "gentle",
		"gifted", "glamorous", "gleaming", "glorious", "good", "gorgeous", "graceful", "grieving", "grotesque", "grumpy", "handsome",
		"happy", "healthy", "helpful", "helpless", "hilarious", "homeless", "homely", "horrible", "hungry", "hurt", "ill", "important",
		"impossible", "inexpensive", "innocent", "inquisitive", "itchy", "jealous", "jittery", "jolly", "joyous", "kind", "lazy", "light",
		"lively", "lonely", "long", "lovely", "lucky", "magnificent", "misty", "modern", "motionless", "muddy", "mushy", "mysterious",
		"nasty", "naughty", "nervous", "nice", "nutty", "obedient", "obnoxious", "odd", "old-fashioned", "open", "outrageous", "outstanding",
		"panicky", "perfect", "plain", "pleasant", "poised", "poor", "powerful", "precious", "prickly", "proud", "putrid", "puzzled",
		"quaint", "real", "relieved", "repulsive", "rich", "scary", "selfish", "shiny", "shy", "silly", "sleepy", "smiling", "smoggy",
		"sore", "sparkling", "splendid", "spotless", "stormy", "strange", "stupid", "successful", "super", "talented", "tame", "tasty",
		"tender", "tense", "terrible", "thankful", "thoughtful", "thoughtless", "tired", "tough", "troubled", "ugliest", "ugly",
		"uninterested", "unsightly", "unusual", "upset", "uptight", "vast", "victorious", "vivacious", "wandering", "weary", "wicked",
		"wide-eyed", "wild", "witty", "worried", "worrisome", "wrong", "zany", "zealous",
	}

	COLOUR_ADJECTIVES = []string{
		"amber", "amethyst", "apricot", "aqua", "aquamarine", "auburn", "azure", "beige", "black", "blue", "bronze", "brown", "buff",
		"carmine", "celadon", "cerise", "cerulean", "charcoal", "chartreuse", "chocolate", "cinnamon", "copper", "coral", "cream",
		"crimson", "cyan", "dark", "denim", "ebony", "ecru", "eggplant", "emerald", "fuchsia", "gold", "goldenrod", "gray", "green",
		"grey", "hue", "indigo", "ivory", "jade", "jet", "khaki", "lavender", "lemon", "light", "lilac", "lime", "magenta", "mahogany",
		"maroon", "mauve", "mustard", "ocher", "olive", "orange", "orchid", "pale", "pastel", "peach", "periwinkle", "persimmon", "pewter",
		"pink", "primary", "puce", "pumpkin", "purple", "rainbow", "red", "rose", "ruby", "russet", "rust", "saffron", "salmon", "sapphire",
		"scarlet", "secondary", "sepia", "shade", "shamrock", "sienna", "silver", "slate", "spectrum", "tan", "tangerine", "taupe", "teal",
		"terracotta", "thistle", "tint", "tomato", "topaz", "turquoise", "ultramarine", "umber", "vermilion", "violet", "viridian", "wheat",
		"white", "wisteria", "yellow",
	}

	NOUNS = []string{
		"Aardvark", "Aardwolf", "Abyssinian", "Addax", "Affenpinscher", "Agouti", "Aidi", "Ainu", "Airedoodle", "Akbash", "Akita", "Albatross",
		"Albertonectes", "Allosaurus", "Allosaurus", "Alpaca", "Alusky", "Amargasaurus", "Amberjack", "Anaconda", "Anchovies", "Andrewsarchus",
		"Angelfish", "Angelshark", "Anglerfish", "Anhinga", "Anomalocaris", "Ant", "Anteater", "Antelope", "Anteosaurus", "Ape", "Arambourgiania",
		"Arapaima", "Archaeoindris", "Archaeopteryx", "Archaeotherium", "Archerfish", "Arctodus", "Arctotherium", "Argentinosaurus", "Armadillo",
		"Armyworm", "Arsinoitherium", "Arthropleura", "Asp", "Aurochs", "Aussiedoodle", "Aussiedor", "Aussiepom", "Australopithecus", "Avocet",
		"Axolotl", "Aye-aye", "Azawakh", "Babirusa", "Baboon", "Badger", "Balinese", "Bandicoot", "Barb", "Barbet", "Barinasuchus", "Barnacle",
		"Barnevelder", "Barosaurus", "Barracuda", "Barylambda", "Basilosaurus", "Bass", "Bassador", "Bassetoodle", "Bat", "Batfish", "Baya",
		"Bea-Tzu", "Beabull", "Beagador", "Beagle", "Beaglier", "Beago", "Bear", "Beaski", "Beauceron", "Beaver", "Bee", "Bee-Eater", "Beefalo",
		"Beetle", "Bergamasco", "Bernedoodle", "Bichir", "Bichpoo", "Bilby", "Binturong", "Bird", "Birman", "Bison", "Blobfish", "Bloodhound",
		"Blowfly", "Bluefish", "Bluegill", "Boas", "Bobcat", "Bobolink", "Boerboel", "Boggle", "Boiga", "Bombay", "Bonefish", "Bongo", "Bonobo",
		"Booby", "Boomslang", "Borador", "Bordoodle", "Borkie", "Boskimo", "Bowfin", "Boxachi", "Boxador", "Boxerdoodle", "Boxfish", "Boxsky",
		"Boxweiler", "Brachiosaurus", "Briard", "Brittany", "Brontosaurus", "Brug", "Budgerigar", "Buffalo", "Bullboxer", "Bulldog", "Bullfrog",
		"Bullmastiff", "Bullsnake", "Bumblebee", "Burmese", "Butterfly", "Caecilian", "Caiman", "Camel", "Cantil", "Canvasback", "Capuchin", "Capybara",
		"Caracal", "Cardinal", "Caribou", "Carp", "Cascabel", "Cassowary", "Cat", "Caterpillar", "Catfish", "Cavador", "Cavapoo", "Centipede",
		"Cephalaspis", "Ceratopsian", "Ceratosaurus", "Chameleon", "Chamois", "Chartreux", "Cheagle", "Cheetah", "Chickadee", "Chicken", "Chigger",
		"Chihuahua", "Chilesaurus", "Chimaera", "Chimpanzee", "Chinchilla", "Chinook", "Chipit", "Chipmunk", "Chipoo", "Chiton", "Chiweenie", "Chorkie",
		"Chusky", "Cicada", "Cichlid", "Clownfish", "Coati", "Cobras", "Cockalier", "Cockapoo", "Cockatiel", "Cockatoo", "Cockle", "Cockroach",
		"Codfish", "Coelacanth", "Collie", "Compsognathus", "Conure", "Copperhead", "Coral", "Corella", "Corgidor", "Corgipoo", "Corkie", "Cormorant",
		"Coryphodon", "Cottonmouth", "Cougar", "Cow", "Coyote", "Crab", "Crane", "Crayfish", "Cricket", "Crocodile", "Crocodylomorph", "Crow",
		"Cryolophosaurus", "Cuckoo", "Cuttlefish", "Dachsador", "Dachshund", "Daeodon", "Dalmadoodle", "Dalmador", "Dalmatian", "Damselfish", "Daniff",
		"Danios", "Daug", "Deer", "Deinocheirus", "Deinosuchus", "Desmostylus", "Dhole", "Dickcissel", "Dickinsonia", "Dik-Dik", "Dilophosaurus",
		"Dimetrodon", "Dingo", "Dinocrocuta", "Dinofelis", "Dinopithecus", "Dinosaurs", "Diplodocus", "Diprotodon", "Discus", "Dobsonfly", "Dodo",
		"Doedicurus", "Dog", "Dolphin", "Donkey", "Dorgi", "Dorkie", "Dormouse", "Douc", "Doxiepoo", "Doxle", "Dragonfish", "Dragonfly",
		"Dreadnoughtus", "Drever", "Duck", "Dugong", "Dunker", "Dunkleosteus", "Dunnock", "Eagle", "Earthworm", "Earwig", "Echidna", "Eel", "Eelpout",
		"Egret", "Eider", "Eland", "Elasmosaurus", "Elasmotherium", "Elephant", "Elk", "Embolotherium", "Emu", "Epidexipteryx", "Ermine", "Eryops",
		"Escolar", "Eskipoo", "Euoplocephalus", "Eurasier", "Eurypterus", "Fairy-Wren", "Falcon", "Fangtooth", "Feist", "Ferret", "Finch", "Firefly",
		"Fish", "Fisher", "Flamingo", "Flea", "Flounder", "Fly", "Flycatcher", "Fossa", "Fox", "Frenchton", "Frengle", "Frigatebird", "Frog", "Frogfish",
		"Frug", "Gadwall", "Gar", "Gastornis", "Gazelle", "Gecko", "Genet", "Gerbil", "Gharial", "Gibbon", "Gigantopithecus", "Giraffe", "Glechon",
		"Glowworm", "Gnat", "Goat", "Goberian", "Goldador", "Goldcrest", "Goldendoodle", "Goldfish", "Gollie", "Gomphotherium", "Goose", "Gopher",
		"Goral", "Gorgosaurus", "Gorilla", "Goshawk", "Gourami", "Grasshopper", "Grebe", "Greyhound", "Griffonshire", "Groenendael", "Grouper", "Grouse",
		"Grunion", "Guppy", "Haddock", "Hagfish", "Haikouichthys", "Hainosaurus", "Halibut", "Hallucigenia", "Hamster", "Hare", "Harrier", "Hartebeest",
		"Hatzegopteryx", "Havamalt", "Havanese", "Havapoo", "Havashire", "Havashu", "Hawk", "Hedgehog", "Helicoprion", "Hellbender", "Heron", "Herrerasaurus",
		"Herring", "Himalayan", "Hippopotamus", "Hogfish", "Hokkaido", "Hoopoe", "Horgi", "Hornbill", "Hornet", "Horse", "Horsefly", "Housefly", "Hovasaurus",
		"Hovawart", "Human", "Hummingbird", "Huntaway", "Huskador", "Huskita", "Husky", "Huskydoodle", "Hyaenodon", "Hyena", "Ibex", "Ibis", "Icadyptes",
		"Ichthyosaurus", "Ichthyostega", "Iguana", "Iguanodon", "Impala", "Inchworm", "Indri", "Insect", "Insects", "Jabiru", "Jacana", "Jack-Chi", "Jackabee",
		"Jackal", "Jackdaw", "Jackrabbit", "Jagdterrier", "Jaguar", "Javanese", "Jellyfish", "Jerboa", "Junglefowl", "Kagu", "Kakapo", "Kangaroo", "Katydid",
		"Kea", "Keagle", "Keelback", "Keeshond", "Kestrel", "Kiang", "Killdeer", "Killifish", "Kingfisher", "Kingklip", "Kinkajou", "Kishu", "Kiwi",
		"Klipspringer", "Knifefish", "Koala", "Kodkod", "Komondor", "Kooikerhondje", "Koolie", "Kouprey", "Kowari", "Krait", "Krill", "Kudu", "Kuvasz",
		"Labahoula", "Labmaraner", "Labrabull", "Labradane", "Labradoodle", "Labraheeler", "Labrottie", "Ladybug", "Ladyfish", "Lamprey", "Lancetfish",
		"Leech", "Leedsichthys", "Lemming", "Lemur", "Leonberger", "Leopard", "Leptocephalus", "Lhasapoo", "Liger", "Limpet", "Linnet", "Lion", "Lionfish",
		"Liopleurodon", "Liopleurodon", "Livyatan", "Lizard", "Lizardfish", "Llama", "Loach", "Lobster", "Locust", "Lorikeet", "Loris", "Lowchen", "Lumpfish",
		"Lungfish", "Lurcher", "Lynx", "Lyrebird", "Lystrosaurus", "Macaque", "Macaw", "Machaeroides", "Macrauchenia", "Maggot", "Magpie", "Magyarosaurus",
		"Maiasaura", "Malchi", "Mallard", "Malteagle", "Maltese", "Maltipom", "Maltipoo", "Mamba", "Manatee", "Mandrill", "Margay", "Markhor", "Marmoset",
		"Marmot", "Masiakasaurus", "Massasauga", "Mastador", "Mastiff", "Mauzer", "Mayfly", "Meagle", "Mealybug", "Meerkat", "Megalania", "Megalochelys",
		"Megalodon", "Meganeura", "Megatherium", "Meiolania", "Merganser", "Microraptor", "Miki", "Milkfish", "Millipede", "Mink", "Mockingbird", "Mojarra",
		"Mole", "Mollusk", "Molly", "Mongoose", "Mongrel", "Monkey", "Monkfish", "Moorhen", "Moose", "Morkie", "Mosasaurus", "Mosquito", "Moth", "Mouse",
		"Mudi", "Mudpuppy", "Mudskipper", "Mule", "Muntjac", "Muskox", "Muskrat", "Muttaburrasaurus", "Nabarlek", "Naegleria", "Narwhal", "Natterjack",
		"Nautilus", "Neanderthal", "Nebelung", "Needlefish", "Nematode", "Newfoundland", "Newfypoo", "Newt", "Nightingale", "Nightjar", "Nilgai",
		"Norrbottenspets", "Nudibranch", "Numbat", "Nuralagus", "Nuthatch", "Nutria", "Nyala", "Oarfish", "Ocelot", "Octopus", "Oilfish", "Okapi", "Olingo",
		"Olm", "Onager", "Opabinia", "Opah", "Opossum", "Orangutan", "Ori-Pei", "Oribi", "Ornithocheirus", "Ornithomimus", "Osprey", "Ostracod", "Ostrich",
		"Otter", "Otterhound", "Ovenbird", "Oviraptor", "Owl", "Ox", "Oxpecker", "Oyster", "Pachycephalosaurus", "Paddlefish", "Pademelon", "Palaeophis",
		"Paleoparadoxia", "Pangolin", "Panther", "Papillon", "Parakeet", "Parasaurolophus", "Parrot", "Parrotfish", "Parrotlet", "Partridge", "Patagotitan",
		"Peacock", "Peagle", "Peekapoo", "Pekingese", "Pelagornis", "Pelagornithidae", "Pelican", "Pelycosaurs", "Penguin", "Persian", "Pheasant", "Phorusrhacos",
		"Phytosaurs", "Pig", "Pigeon", "Pika", "Pinfish", "Pipefish", "Piranha", "Pitador", "Pitsky", "Platybelodon", "Platypus", "Plesiosaur", "Pliosaur",
		"Pointer", "Polacanthus", "Polecat", "Pomapoo", "Pomchi", "Pomeagle", "Pomeranian", "Pomsky", "Poochon", "Poodle", "Poogle", "Porcupine", "Porcupinefish",
		"Possum", "Potoo", "Potoroo", "Prawn", "Procoptodon", "Pronghorn", "Psittacosaurus", "Pteranodon", "Pterodactyl", "Pudelpointer", "Puertasaurus",
		"Pufferfish", "Puffin", "Pug", "Pugapoo", "Puggle", "Pugshire", "Puli", "Puma", "Pumi", "Purussaurus", "Pyrador", "Pyredoodle", "Pyrosome", "Python",
		"Quagga", "Quail", "Quetzal", "Quokka", "Quoll", "Rabbit", "Raccoon", "Ragamuffin", "Ragdoll", "Raggle", "Rat", "Rattlesnake", "Redstart", "Reindeer",
		"Repenomamus", "Rhamphosuchus", "Rhea", "Rhinoceros", "Roadrunner", "Robin", "Rockfish", "Rodents", "Rooster", "Rotterman", "Rottle", "Rottsky",
		"Rottweiler", "Sable", "Saiga", "Sailfish", "Salamander", "Salmon", "Saluki", "Sambar", "Samoyed", "Sandpiper", "Sandworm", "Saola", "Sapsali",
		"Sarcosuchus", "Sardines", "Sarkastodon", "Sarplaninac", "Sauropoda", "Sauropoda", "Sawfish", "Scallops", "Schapendoes", "Schipperke", "Schneagle",
		"Schnoodle", "Scorpion", "Sculpin", "Scutosaurus", "Seagull", "Seahorse", "Seal", "Serval", "Seymouria", "Shantungosaurus", "Shark", "Shastasaurus",
		"Sheep", "Sheepadoodle", "Shepadoodle", "Shepkita", "Shepweiler", "Shichi", "Shikoku", "Shiranian", "Shollie", "Shrew", "Shrimp", "Siamese", "Siberian",
		"Siberpoo", "Sidewinder", "Simbakubwa", "Sinosauropteryx", "Sivatherium", "Skua", "Skunk", "Sloth", "Slug", "Smilosuchus", "Snail", "Snailfish", "Snake",
		"Snorkie", "Snowshoe", "Somali", "Spalax", "Spanador", "Sparrow", "Sparrowhawk", "Sphynx", "Spider", "Spinosaurus", "Sponge", "Springador", "Springbok",
		"Springerdoodle", "Squid", "Squirrel", "Squirrelfish", "Stabyhoun", "Starfish", "Stingray", "Stoat", "Stonechat", "Stonefish", "Stork", "Stromatolite",
		"Stupendemys", "Sturgeon", "Styracosaurus", "Suchomimus", "Suckerfish", "Supersaurus", "Superworm", "Surgeonfish", "Swallow", "Swan", "Swordfish", "Taipan",
		"Takin", "Tamarin", "Tamaskan", "Tang", "Tapir", "Tarantula", "Tarbosaurus", "Tarpon", "Tarsier", "Tenrec", "Termite", "Terrier", "Tetra", "Thalassomedon",
		"Thanatosdrakon", "Therizinosaurus", "Theropod", "Thrush", "Thylacoleo", "Thylacosmilus", "Tick", "Tiffany", "Tiger", "Tiktaalik", "Titanoboa", "Titanosaur",
		"Toadfish", "Torkie", "Tornjak", "Tortoise", "Tosa", "Toucan", "Towhee", "Toxodon", "Treecreeper", "Treehopper", "Triggerfish", "Troodon", "Tropicbird",
		"Trout", "Tuatara", "Tuna", "Turaco", "Turkey", "Turnspit", "Turtles", "Tusoteuthis", "Tylosaurus", "Uakari", "Uguisu", "Uintatherium", "Umbrellabird",
		"Urial", "Utonagan", "Vaquita", "Veery", "Vegavis", "Velociraptor", "Vicuña", "Vinegaroon", "Viper", "Viperfish", "Vizsla", "Vole", "Vulture", "Waimanu",
		"Wallaby", "Walrus", "Warbler", "Warthog", "Wasp", "Waterbuck", "Weasel", "Weimaraner", "Weimardoodle", "Westiepoo", "Whimbrel", "Whinchat", "Whippet",
		"Whiting", "Whoodle", "Wildebeest", "Wiwaxia", "Wolf", "Wolffish", "Wolverine", "Wombat", "Woodlouse", "Woodpecker", "Woodrat", "Worm", "Wrasse", "Wryneck",
		"Xenacanthus", "Xenoceratops", "Xenoposeidon", "Xenotarsosaurus", "Xerus", "Xiaosaurus", "Xiaotingia", "Xiongguanlong", "Xiphactinus", "Xoloitzcuintli",
		"Yabby", "Yak", "Yarara", "Yellowhammer", "Yellowthroat", "Yoranian", "Yorkiepoo", "Zebra", "Zebu", "Zokor", "Zonkey", "Zorse", "Zuchon",
	}
)

func refreshSeed() {
	rnd.Seed(time.Now().UTC().UnixNano())
}

func GenerateRandomTwoPartName() string {
	refreshSeed()

	opinion := OPINION_ADJECTIVES[rnd.Intn(len(OPINION_ADJECTIVES))]
	noun := NOUNS[rnd.Intn(len(NOUNS))]

	return strings.ToLower(fmt.Sprintf("%s-%s", opinion, noun))
}

func GenerateRandomThreePartName() string {
	refreshSeed()

	opinion := OPINION_ADJECTIVES[rnd.Intn(len(OPINION_ADJECTIVES))]
	colour := COLOUR_ADJECTIVES[rnd.Intn(len(COLOUR_ADJECTIVES))]
	noun := NOUNS[rnd.Intn(len(NOUNS))]

	return strings.ToLower(fmt.Sprintf("%s-%s-%s", opinion, colour, noun))
}
